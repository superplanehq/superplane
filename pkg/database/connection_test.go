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
