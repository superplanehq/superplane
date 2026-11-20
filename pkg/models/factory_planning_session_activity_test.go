package models

import (
	"context"
	"database/sql"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPlanningSessionActivityConflictPreservesOwnership(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	db := newPlanningActivityTestDatabase(ctx, t)
	callerDB := pinPlanningActivityTestConnection(ctx, t, db)
	ownerDB := pinPlanningActivityTestConnection(ctx, t, db)
	observer := pinPlanningActivityTestConnection(ctx, t, db)
	caller := FactoryPlanningSession{ID: uuid.New()}
	owner := FactoryPlanningSession{ID: uuid.New()}
	require.NoError(t, observer.Exec("INSERT INTO factory_planning_sessions (id) VALUES (?), (?)", caller.ID, owner.ID).Error)

	now := time.Now().UTC().Truncate(time.Microsecond)
	completedAt := now.Add(time.Second)
	activity := PlanningSessionActivity{
		ID: uuid.New(), SessionID: owner.ID, SchemaVersion: 2,
		Provider: "owner-provider", Status: PlanningSessionActivityStatusPassed, LastSequence: 1,
		Snapshot: datatypes.NewJSONType(PlanningSessionActivitySnapshot{
			SchemaVersion: 2, Provider: "owner-provider", Status: PlanningSessionActivityStatusPassed,
			Sequence: 1, ActivityID: "owner-snapshot", Items: []PlanningSessionActivityItem{{Text: "owner content"}},
		}),
		StartedAt: now, CompletedAt: &completedAt, CreatedAt: now, UpdatedAt: now,
	}
	foreign := activity
	foreign.SessionID = caller.ID
	foreign.SchemaVersion = 3
	foreign.Provider = "caller-provider"
	foreign.Status = PlanningSessionActivityStatusFailed
	foreign.LastSequence = 100
	foreign.Snapshot = datatypes.NewJSONType(PlanningSessionActivitySnapshot{
		SchemaVersion: 3, Provider: "caller-provider", Status: PlanningSessionActivityStatusFailed,
		Sequence: 100, ActivityID: "caller-snapshot", Items: []PlanningSessionActivityItem{{Text: "foreign content"}},
	})
	foreign.StartedAt = now.Add(time.Minute)
	foreign.CompletedAt = nil
	foreign.CreatedAt = now.Add(time.Minute)
	foreign.UpdatedAt = now.Add(time.Minute)

	var callerPID, ownerPID int
	require.NoError(t, callerDB.Raw("SELECT pg_backend_pid()").Scan(&callerPID).Error)
	require.NoError(t, ownerDB.Raw("SELECT pg_backend_pid()").Scan(&ownerPID).Error)
	ownerTx := ownerDB.Begin(&sql.TxOptions{Isolation: sql.LevelReadCommitted})
	require.NoError(t, ownerTx.Error)
	t.Cleanup(func() { ownerTx.Rollback() })
	require.NoError(t, ownerTx.Create(&activity).Error)
	var baseline PlanningSessionActivity
	require.NoError(t, ownerTx.First(&baseline, "id = ?", activity.ID).Error)

	type upsertResult struct {
		written bool
		err     error
	}
	result := make(chan upsertResult, 1)
	done := make(chan struct{})
	t.Cleanup(func() {
		ownerTx.Rollback()
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("activity upsert did not stop during cleanup")
		}
	})
	go func() {
		defer close(done)
		written, err := caller.UpsertActivity(callerDB, foreign)
		result <- upsertResult{written, err}
	}()

	// The insert lock proves the ownership SELECT missed B's uncommitted row.
	require.Eventually(t, func() bool {
		var blocked bool
		err := observer.Raw(`SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity
			WHERE pid = ? AND datname = current_database() AND wait_event_type = 'Lock'
			AND query LIKE 'INSERT INTO "factory_planning_session_activities"%'
			AND ? = ANY(pg_blocking_pids(pid))
		)`, callerPID, ownerPID).Scan(&blocked).Error
		return err == nil && blocked
	}, 5*time.Second, 10*time.Millisecond, "caller must wait on the owner's insert conflict")
	require.NoError(t, ownerTx.Commit().Error)
	select {
	case attempted := <-result:
		require.NoError(t, attempted.err)
		assert.False(t, attempted.written, "another session must not write on conflict")
	case <-ctx.Done():
		t.Fatal("activity upsert did not complete: ", ctx.Err())
	}
	readActivity := func() PlanningSessionActivity {
		t.Helper()
		var stored PlanningSessionActivity
		require.NoError(t, observer.First(&stored, "id = ?", activity.ID).Error)
		return stored
	}
	assert.Equal(t, baseline, readActivity(), "foreign conflict must preserve every persisted field")

	newer := foreign
	newer.SessionID = owner.ID
	newer.Provider = "owner-updated-provider"
	newer.LastSequence = 2 // Below the foreign sequence, so the race cannot suppress progress.
	newer.StartedAt = baseline.StartedAt.Add(time.Minute)
	newer.UpdatedAt = baseline.UpdatedAt.Add(time.Minute)
	snapshot := newer.Snapshot.Data()
	snapshot.Provider = newer.Provider
	snapshot.Items = []PlanningSessionActivityItem{{Text: "new owner content"}}
	snapshot.Sequence = 2
	newer.Snapshot = datatypes.NewJSONType(snapshot)
	written, err := owner.UpsertActivity(ownerDB, newer)
	require.NoError(t, err)
	require.True(t, written)
	newer.CreatedAt = baseline.CreatedAt // Creation time is not an update column.
	require.Equal(t, newer, readActivity())
	for _, sequence := range []int64{1, 2} {
		stale := activity
		stale.LastSequence = sequence
		written, err = owner.UpsertActivity(ownerDB, stale)
		require.NoError(t, err)
		assert.False(t, written)
		assert.Equal(t, newer, readActivity())
	}

	// A foreign activity visible before the ownership SELECT retains the precheck error.
	written, err = caller.UpsertActivity(callerDB, foreign)
	assert.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)
	assert.False(t, written)
	assert.Equal(t, newer, readActivity())
}

func newPlanningActivityTestDatabase(ctx context.Context, t *testing.T) *sql.DB {
	t.Helper()
	connectionURL := url.URL{
		Scheme: "postgres", Host: net.JoinHostPort(os.Getenv("DB_HOST"), os.Getenv("DB_PORT")),
		Path: "/" + os.Getenv("DB_NAME"), User: url.UserPassword(os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD")),
	}
	sslMode := "disable"
	if os.Getenv("POSTGRES_DB_SSL") == "true" {
		sslMode = "require"
	}
	options := url.Values{
		"sslmode": {sslMode},
		"options": {"-c statement_timeout=10000 -c lock_timeout=10000 -c idle_in_transaction_session_timeout=15000 -c default_transaction_isolation=read\\ committed"},
	}
	connectionURL.RawQuery = options.Encode()
	open := func() *sql.DB {
		t.Helper()
		pool, err := sql.Open("pgx", connectionURL.String())
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, pool.Close()) })
		require.NoError(t, pool.PingContext(ctx))
		return pool
	}
	admin := open()
	adminDB := pinPlanningActivityTestConnection(ctx, t, admin)
	require.NoError(t, database.VerifyTestDatabase(adminDB))
	name := "planning_activity_" + strings.ReplaceAll(uuid.NewString(), "-", "") + "_test"
	_, err := admin.ExecContext(ctx, `CREATE DATABASE "`+name+`"`)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := admin.ExecContext(cleanupCtx, `DROP DATABASE "`+name+`"`)
		assert.NoError(t, err)
	})
	connectionURL.Path = "/" + name
	pool := open()
	fixture := pinPlanningActivityTestConnection(ctx, t, pool)
	require.NoError(t, database.VerifyTestDatabase(fixture))
	// Only the session key is needed; activity types and constraints match the migration.
	require.NoError(t, fixture.Exec("CREATE TABLE factory_planning_sessions (id UUID PRIMARY KEY)").Error)
	require.NoError(t, fixture.Exec(`
		CREATE TABLE factory_planning_session_activities (
			id UUID PRIMARY KEY,
			session_id UUID NOT NULL REFERENCES factory_planning_sessions(id) ON DELETE CASCADE,
			schema_version INTEGER NOT NULL, provider TEXT NOT NULL, status TEXT NOT NULL,
			last_sequence BIGINT NOT NULL, snapshot JSONB NOT NULL,
			started_at TIMESTAMPTZ NOT NULL, completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`).Error)
	return pool
}

func pinPlanningActivityTestConnection(ctx context.Context, t *testing.T, pool *sql.DB) *gorm.DB {
	t.Helper()
	conn, err := pool.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	return db.WithContext(ctx)
}
