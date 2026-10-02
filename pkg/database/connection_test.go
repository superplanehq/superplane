package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	postgresdrv "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgresDialectorPrefersSimpleProtocol(t *testing.T) {
	dialector := postgresDialector("postgres://user:pass@db.example:5432/appdb?sslmode=disable")
	pgDialector, ok := dialector.(*postgresdrv.Dialector)
	require.True(t, ok)
	require.NotNil(t, pgDialector.Config)
	require.True(t, pgDialector.PreferSimpleProtocol)
}

func TestReadSucceedsAfterIntegerColumnBecomesBigint(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set (run with make test in Docker)")
	}

	preparedStatementDSN := withQueryExecMode(t, postgresDSNFromEnv(), "cache_statement")
	db, err := gorm.Open(postgresDialector(preparedStatementDSN), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	_, err = conn.ExecContext(ctx, `CREATE TEMPORARY TABLE column_type_change (id integer PRIMARY KEY, amount integer)`)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `INSERT INTO column_type_change (id, amount) VALUES (1, 7)`)
	require.NoError(t, err)

	const query = `SELECT amount FROM column_type_change WHERE id = $1`
	var amount int64
	require.NoError(t, conn.QueryRowContext(ctx, query, 1).Scan(&amount))
	require.Equal(t, int64(7), amount)

	_, err = conn.ExecContext(ctx, `ALTER TABLE column_type_change ALTER COLUMN amount TYPE bigint`)
	require.NoError(t, err)

	require.NoError(t, conn.QueryRowContext(ctx, query, 1).Scan(&amount))
	require.Equal(t, int64(7), amount)
}

func TestBuildPostgresDSN_sessionTimeouts(t *testing.T) {
	dsn := buildPostgresDSN(DSNConfig{
		Host:            "db.example",
		Port:            "5432",
		Name:            "appdb",
		User:            "u",
		Pass:            "p",
		Ssl:             "disable",
		ApplicationName: "testapp",
	}, 60*time.Second, 30*time.Second)
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	opts := q.Get("options")
	if !strings.Contains(opts, "statement_timeout=60000") {
		t.Fatalf("dsn options: %q", opts)
	}
	if !strings.Contains(opts, "idle_in_transaction_session_timeout=30000") {
		t.Fatalf("dsn options: %q", opts)
	}
	if q.Get("default_query_exec_mode") != "describe_exec" {
		t.Fatalf("dsn default_query_exec_mode: %q", q.Get("default_query_exec_mode"))
	}
}

func TestDSNConfigFromEnv(t *testing.T) {
	t.Setenv("DB_HOST", "db.example")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "superplane_test")
	t.Setenv("DB_USERNAME", "postgres")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("POSTGRES_DB_SSL", "true")
	t.Setenv("APPLICATION_NAME", "superplane-test")

	cfg := dsnConfigFromEnv()

	require.Equal(t, "db.example", cfg.Host)
	require.Equal(t, "5432", cfg.Port)
	require.Equal(t, "superplane_test", cfg.Name)
	require.Equal(t, "postgres", cfg.User)
	require.Equal(t, "secret", cfg.Pass)
	require.Equal(t, "require", cfg.Ssl)
	require.Equal(t, "superplane-test", cfg.ApplicationName)
}

func TestOpenDedicatedSQLDB_ConfiguresDedicatedPool(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set (run with make test in Docker)")
	}

	db, err := OpenDedicatedSQLDB("agent-stream-lock-test", 0)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	stats := db.Stats()
	require.Equal(t, 1, stats.MaxOpenConnections)
	require.NoError(t, db.Ping())
}

func TestIsTestDatabaseName(t *testing.T) {
	require.True(t, isTestDatabaseName("superplane_test"))
	require.False(t, isTestDatabaseName("superplane_dev"))
	require.False(t, isTestDatabaseName("superplane"))
	require.False(t, isTestDatabaseName("superplane_test_backup_dev"))
}

func TestVerifyTestDatabase(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set (run with make test in Docker)")
	}

	require.NoError(t, VerifyTestDatabase(Conn()))
}

func TestClassifyPostgresSessionTimeout(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want postgresTimeoutKind
	}{
		{
			"statement_timeout",
			&pgconn.PgError{Message: "canceling statement due to statement timeout"},
			postgresTimeoutStatement,
		},
		{
			"idle_in_transaction",
			&pgconn.PgError{Message: "terminating connection due to idle-in-transaction timeout"},
			postgresTimeoutIdleInTransaction,
		},
		{"unrelated", errors.New("duplicate key value"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyPostgresSessionTimeout(tt.err); got != tt.want {
				t.Fatalf("classifyPostgresSessionTimeout() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPostgres_statementTimeoutEnforced(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set (run with make test in Docker)")
	}

	t.Setenv("DB_STATEMENT_TIMEOUT", "100ms")
	t.Setenv("DB_IDLE_IN_TRANSACTION_SESSION_TIMEOUT", "10s")

	sslMode := "disable"
	if os.Getenv("POSTGRES_DB_SSL") == "true" {
		sslMode = "require"
	}

	c := DSNConfig{
		Host:            os.Getenv("DB_HOST"),
		Port:            os.Getenv("DB_PORT"),
		Name:            os.Getenv("DB_NAME"),
		User:            os.Getenv("DB_USERNAME"),
		Pass:            os.Getenv("DB_PASSWORD"),
		Ssl:             sslMode,
		ApplicationName: os.Getenv("APPLICATION_NAME"),
	}

	cfg := LoadConfig()
	dsn := buildPostgresDSN(c, cfg.StatementTimeout, cfg.IdleInTransactionSessionTimeout)

	db, err := gorm.Open(postgresdrv.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB.Close()
	})

	err = db.Exec("SELECT pg_sleep(0.25)").Error
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "statement timeout")

	require.NoError(t, db.Exec("SELECT 1").Error)
}

func TestParameterizedReadAfterColumnAdd(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set (run with make test in Docker)")
	}

	productionDSN := postgresDSNFromEnv()

	t.Run("cache statement", func(t *testing.T) {
		dsn := withQueryExecMode(t, productionDSN, "cache_statement")
		_, err := repeatParameterizedReadAfterColumnAdd(t, dsn)
		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "0A000", pgErr.Code)
	})

	t.Run("production dsn", func(t *testing.T) {
		columns, err := repeatParameterizedReadAfterColumnAdd(t, productionDSN)
		require.NoError(t, err)
		require.Equal(t, []string{"id", "extra"}, columns)
	})
}

func postgresDSNFromEnv() string {
	cfg := LoadConfig()
	return buildPostgresDSN(dsnConfigFromEnv(), cfg.StatementTimeout, cfg.IdleInTransactionSessionTimeout)
}

func withQueryExecMode(t *testing.T, dsn, mode string) string {
	t.Helper()
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	query := u.Query()
	query.Set("default_query_exec_mode", mode)
	u.RawQuery = query.Encode()
	return u.String()
}

func repeatParameterizedReadAfterColumnAdd(t *testing.T, dsn string) ([]string, error) {
	t.Helper()
	conn := openPinnedSQLConn(t, dsn)
	table := createScratchTable(t, conn)
	ctx := context.Background()
	query := "SELECT * FROM " + table + " WHERE id = $1"

	columns, err := readScratchColumns(ctx, conn, query, 1)
	require.NoError(t, err)
	require.Equal(t, []string{"id"}, columns)

	_, err = conn.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN extra text")
	require.NoError(t, err)

	return readScratchColumns(ctx, conn, query, 1)
}

func openPinnedSQLConn(t *testing.T, dsn string) *sql.Conn {
	t.Helper()
	db, err := gorm.Open(postgresdrv.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	conn, err := sqlDB.Conn(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	return conn
}

func createScratchTable(t *testing.T, conn *sql.Conn) string {
	t.Helper()
	name, err := scratchTableName()
	require.NoError(t, err)
	ctx := context.Background()
	_, err = conn.ExecContext(ctx, "CREATE TABLE "+name+" (id integer PRIMARY KEY)")
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, "INSERT INTO "+name+" (id) VALUES (1)")
	require.NoError(t, err)
	t.Cleanup(func() {
		dropScratchTable(t, name)
	})
	return name
}

func scratchTableName() (string, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "query_exec_mode_" + hex.EncodeToString(raw[:]), nil
}

func dropScratchTable(t *testing.T, name string) {
	t.Helper()
	db, err := gorm.Open(postgresdrv.Open(postgresDSNFromEnv()), &gorm.Config{})
	if err != nil {
		t.Errorf("open cleanup connection: %v", err)
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Errorf("cleanup sql.DB: %v", err)
		return
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec("DROP TABLE IF EXISTS " + name); err != nil {
		t.Errorf("drop scratch table %s: %v", name, err)
	}
}

func readScratchColumns(ctx context.Context, conn *sql.Conn, query string, id int) ([]string, error) {
	rows, err := conn.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return columns, nil
}
