package database

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresTimeoutDetails_extractsContext(t *testing.T) {
	begin := time.Now().Add(-150 * time.Millisecond)
	pgErr := &pgconn.PgError{
		Code:    "25P03",
		Message: "terminating connection due to idle-in-transaction timeout",
	}
	fc := func() (string, int64) {
		return "UPDATE workflow_versions SET nodes = $1 WHERE id = $2", 1
	}

	d := newPostgresTimeoutDetails(pgErr, postgresTimeoutIdleInTransaction, begin, fc)

	require.Equal(t, postgresTimeoutIdleInTransaction, d.kind)
	require.Equal(t, "25P03", d.sqlState)
	require.Equal(t, pgErr.Message, d.pgMessage)
	require.Equal(t, "UPDATE workflow_versions SET nodes = $1 WHERE id = $2", d.sql)
	require.Equal(t, int64(1), d.rowsAffected)
	require.GreaterOrEqual(t, d.duration, 100*time.Millisecond)
	require.NotEmpty(t, d.caller.function, "expected caller function to be captured")
	require.NotContains(t, d.caller.function, "/pkg/database.", "caller should be outside the database package")
}

func TestNewPostgresTimeoutDetails_truncatesLongSQL(t *testing.T) {
	long := strings.Repeat("x", 5000)
	d := newPostgresTimeoutDetails(
		errors.New("statement timeout"),
		postgresTimeoutStatement,
		time.Now(),
		func() (string, int64) { return long, 0 },
	)
	require.LessOrEqual(t, len(d.sql), 2048)
	require.True(t, strings.HasSuffix(d.sql, "..."), "long SQL should be truncated with ellipsis")
}

func TestNewPostgresTimeoutDetails_handlesNilFc(t *testing.T) {
	d := newPostgresTimeoutDetails(
		errors.New("statement timeout"),
		postgresTimeoutStatement,
		time.Now(),
		nil,
	)
	require.Equal(t, "", d.sql)
	require.Equal(t, int64(0), d.rowsAffected)
}

func TestPostgresTimeoutDetails_fingerprintIncludesKindAndCaller(t *testing.T) {
	d := postgresTimeoutDetails{
		kind:   postgresTimeoutIdleInTransaction,
		caller: callerFrame{function: "github.com/example/foo.Bar"},
	}
	fp := d.fingerprint()
	require.Equal(t, []string{
		"postgres_session_timeout",
		string(postgresTimeoutIdleInTransaction),
		"github.com/example/foo.Bar",
	}, fp)
}

func TestPostgresTimeoutDetails_fingerprintFallsBackWhenCallerUnknown(t *testing.T) {
	d := postgresTimeoutDetails{kind: postgresTimeoutStatement}
	fp := d.fingerprint()
	require.Equal(t, "unknown", fp[2])
}

func TestShouldCaptureToSentry_rateLimitsSameFingerprint(t *testing.T) {
	resetSentryRateLimitForTest(t)

	fp := []string{"postgres_session_timeout", "statement_timeout", "test/Caller"}
	require.True(t, shouldCaptureToSentry(fp), "first call should be allowed")
	require.False(t, shouldCaptureToSentry(fp), "second call within window should be rate limited")
}

func TestShouldCaptureToSentry_doesNotRateLimitDifferentFingerprints(t *testing.T) {
	resetSentryRateLimitForTest(t)

	a := []string{"postgres_session_timeout", "statement_timeout", "pkg/A.Func"}
	b := []string{"postgres_session_timeout", "idle_in_transaction_session_timeout", "pkg/A.Func"}
	c := []string{"postgres_session_timeout", "statement_timeout", "pkg/B.Func"}

	require.True(t, shouldCaptureToSentry(a))
	require.True(t, shouldCaptureToSentry(b))
	require.True(t, shouldCaptureToSentry(c))
	require.False(t, shouldCaptureToSentry(a))
}

func TestTruncate(t *testing.T) {
	require.Equal(t, "abc", truncate("abc", 10))
	require.Equal(t, "abc", truncate("abc", 3))
	require.Equal(t, "abcd...", truncate("abcdefghij", 7))
	require.Equal(t, "ab", truncate("abcdef", 2))
	require.Equal(t, "abcdef", truncate("abcdef", 0))
}

func resetSentryRateLimitForTest(t *testing.T) {
	t.Helper()
	sentryCaptureRateLimit.mu.Lock()
	prev := sentryCaptureRateLimit.lastSent
	sentryCaptureRateLimit.lastSent = make(map[string]time.Time)
	sentryCaptureRateLimit.mu.Unlock()
	t.Cleanup(func() {
		sentryCaptureRateLimit.mu.Lock()
		sentryCaptureRateLimit.lastSent = prev
		sentryCaptureRateLimit.mu.Unlock()
	})
}
