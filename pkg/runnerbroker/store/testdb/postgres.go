package testdb

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/superplanehq/superplane/pkg/runnerbroker/store"
)

var testMu sync.Mutex

// Open returns a PostgresStore backed by TEST_DATABASE_URL.
func Open(t *testing.T) (*store.PostgresStore, func()) {
	t.Helper()
	testMu.Lock()
	t.Cleanup(func() { testMu.Unlock() })

	dsn := requireDSN(t)

	st, err := store.OpenPostgres(dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := st.Truncate(ctx); err != nil {
		_ = st.Close()
		t.Fatalf("truncate postgres: %v", err)
	}
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = st.Truncate(ctx)
		_ = st.Close()
	}
	return st, cleanup
}

// DSN returns a PostgreSQL DSN for subprocess tests (task-broker binary).
func DSN(t *testing.T) string {
	t.Helper()
	return requireDSN(t)
}

func requireDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset — start Postgres locally or use CI sem-service (see README)")
	}
	return dsn
}
