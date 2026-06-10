package database

import (
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
