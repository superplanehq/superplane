package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	gormlogger "gorm.io/gorm/logger"
)

type gormTimeoutLogger struct {
	base gormlogger.Interface
}

func newGormTimeoutLogger(base gormlogger.Interface) gormlogger.Interface {
	return &gormTimeoutLogger{base: base}
}

func (w *gormTimeoutLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &gormTimeoutLogger{base: w.base.LogMode(level)}
}

func (w *gormTimeoutLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	w.base.Info(ctx, msg, data...)
}

func (w *gormTimeoutLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	w.base.Warn(ctx, msg, data...)
}

func (w *gormTimeoutLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	w.base.Error(ctx, msg, data...)
}

func (w *gormTimeoutLogger) Trace(
	ctx context.Context,
	begin time.Time,
	fc func() (sql string, rowsAffected int64),
	err error,
) {
	if err != nil {
		kind := classifyPostgresSessionTimeout(err)
		if kind != "" {
			details := newPostgresTimeoutDetails(err, kind, begin, fc)
			logPostgresSessionTimeout(details)
			capturePostgresSessionTimeoutToSentry(ctx, details)
		}
	}
	w.base.Trace(ctx, begin, fc, err)
}

type postgresTimeoutKind string

const (
	postgresTimeoutStatement         postgresTimeoutKind = "statement_timeout"
	postgresTimeoutIdleInTransaction postgresTimeoutKind = "idle_in_transaction_session_timeout"
)

// postgresTimeoutDetails captures the bits of context we want to surface
// alongside a Postgres session timeout error. We collect them up front so
// the Sentry capture path does not depend on the GORM trace closure being
// safe to call after the fact.
type postgresTimeoutDetails struct {
	err          error
	kind         postgresTimeoutKind
	sqlState     string
	pgMessage    string
	sql          string
	rowsAffected int64
	duration     time.Duration
	caller       callerFrame
}

type callerFrame struct {
	function string
	file     string
	line     int
}

func (c callerFrame) String() string {
	if c.function == "" && c.file == "" {
		return "unknown"
	}
	if c.file == "" {
		return c.function
	}
	return fmt.Sprintf("%s (%s:%d)", c.function, c.file, c.line)
}

// fingerprint returns a stable, low-cardinality identifier for grouping
// similar timeouts together in Sentry while still keeping different call
// sites separated.
func (d postgresTimeoutDetails) fingerprint() []string {
	caller := d.caller.function
	if caller == "" {
		caller = "unknown"
	}
	return []string{"postgres_session_timeout", string(d.kind), caller}
}

func newPostgresTimeoutDetails(
	err error,
	kind postgresTimeoutKind,
	begin time.Time,
	fc func() (sql string, rowsAffected int64),
) postgresTimeoutDetails {
	d := postgresTimeoutDetails{
		err:      err,
		kind:     kind,
		duration: time.Since(begin),
		caller:   firstCallerOutsideDatabasePackage(),
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		d.sqlState = pgErr.Code
		d.pgMessage = pgErr.Message
	}

	if fc != nil {
		sql, rows := fc()
		d.sql = truncate(sql, 2048)
		d.rowsAffected = rows
	}

	return d
}

// firstCallerOutsideDatabasePackage walks the stack to find the first frame
// that is not in the database package nor inside GORM/runtime. This is what
// we want to attribute the timeout to in Sentry, because the closure inside
// hub.WithScope and the gorm internals are the same for every event and
// would otherwise collapse all timeouts into a single Sentry issue.
func firstCallerOutsideDatabasePackage() callerFrame {
	const maxDepth = 32
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(2, pcs)
	if n == 0 {
		return callerFrame{}
	}
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "" {
			if !more {
				break
			}
			continue
		}
		if isInfrastructureFrame(frame.Function) {
			if !more {
				break
			}
			continue
		}
		return callerFrame{
			function: frame.Function,
			file:     frame.File,
			line:     frame.Line,
		}
	}
	return callerFrame{}
}

func isInfrastructureFrame(fn string) bool {
	switch {
	case strings.HasPrefix(fn, "github.com/superplanehq/superplane/pkg/database."):
		return true
	case strings.HasPrefix(fn, "gorm.io/"):
		return true
	case strings.HasPrefix(fn, "runtime."):
		return true
	default:
		return false
	}
}

func classifyPostgresSessionTimeout(err error) postgresTimeoutKind {
	if err == nil {
		return ""
	}
	msg := extractPostgresMessage(err)
	ml := strings.ToLower(msg)
	switch {
	case strings.Contains(ml, "statement timeout"):
		return postgresTimeoutStatement
	case strings.Contains(ml, "idle-in-transaction"):
		return postgresTimeoutIdleInTransaction
	case strings.Contains(ml, "idle in transaction"):
		return postgresTimeoutIdleInTransaction
	default:
		return ""
	}
}

func extractPostgresMessage(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Message
	}
	return err.Error()
}

func logPostgresSessionTimeout(d postgresTimeoutDetails) {
	switch d.kind {
	case postgresTimeoutStatement:
		log.Printf(
			"[database] PostgreSQL statement_timeout exceeded after %s at %s: %v",
			d.duration, d.caller, d.err,
		)
	case postgresTimeoutIdleInTransaction:
		log.Printf(
			"[database] PostgreSQL idle_in_transaction_session_timeout exceeded after %s at %s: %v",
			d.duration, d.caller, d.err,
		)
	}
}

// sentryCaptureRateLimit caps how often we send the same fingerprint to
// Sentry. When the database is unhealthy we can hit dozens of timeouts per
// second; without a rate limit each one is its own event and we both flood
// Sentry and pay a per-event allocation cost on every failed query.
var sentryCaptureRateLimit = struct {
	mu       sync.Mutex
	lastSent map[string]time.Time
	window   time.Duration
}{
	lastSent: make(map[string]time.Time),
	window:   time.Minute,
}

func shouldCaptureToSentry(fingerprint []string) bool {
	key := strings.Join(fingerprint, "|")
	now := time.Now()
	sentryCaptureRateLimit.mu.Lock()
	defer sentryCaptureRateLimit.mu.Unlock()
	if last, ok := sentryCaptureRateLimit.lastSent[key]; ok {
		if now.Sub(last) < sentryCaptureRateLimit.window {
			return false
		}
	}
	sentryCaptureRateLimit.lastSent[key] = now
	return true
}

func capturePostgresSessionTimeoutToSentry(ctx context.Context, d postgresTimeoutDetails) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	if hub == nil || hub.Client() == nil {
		return
	}

	fingerprint := d.fingerprint()
	if !shouldCaptureToSentry(fingerprint) {
		return
	}

	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)
		scope.SetFingerprint(fingerprint)
		scope.SetTag("postgres_timeout", string(d.kind))
		if d.sqlState != "" {
			scope.SetTag("postgres_sqlstate", d.sqlState)
		}
		if d.caller.function != "" {
			scope.SetTag("postgres_timeout_caller", d.caller.function)
		}
		extras := map[string]interface{}{
			"duration_ms":   d.duration.Milliseconds(),
			"rows_affected": d.rowsAffected,
		}
		if d.sql != "" {
			extras["sql"] = d.sql
		}
		if d.pgMessage != "" {
			extras["postgres_message"] = d.pgMessage
		}
		if d.caller.file != "" {
			extras["caller_location"] = fmt.Sprintf("%s:%d", d.caller.file, d.caller.line)
		}
		scope.SetExtras(extras)
		hub.CaptureException(d.err)
	})
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
