package database

import (
	"context"
	"errors"
	"log"
	"strings"
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
		logPostgresSessionTimeoutIfMatched(err)
	}
	w.base.Trace(ctx, begin, fc, err)
}

type postgresTimeoutKind string

const (
	postgresTimeoutStatement         postgresTimeoutKind = "statement_timeout"
	postgresTimeoutIdleInTransaction postgresTimeoutKind = "idle_in_transaction_session_timeout"
)

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

func logPostgresSessionTimeoutIfMatched(err error) {
	kind := classifyPostgresSessionTimeout(err)
	if kind == "" {
		return
	}
	switch kind {
	case postgresTimeoutStatement:
		log.Printf("[database] PostgreSQL statement_timeout exceeded: %v", err)
	case postgresTimeoutIdleInTransaction:
		log.Printf("[database] PostgreSQL idle_in_transaction_session_timeout exceeded: %v", err)
	}
	capturePostgresSessionTimeoutToSentry(err, kind)
}

func capturePostgresSessionTimeoutToSentry(err error, kind postgresTimeoutKind) {
	hub := sentry.CurrentHub()
	if hub == nil || hub.Client() == nil {
		return
	}
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("postgres_timeout", string(kind))
		hub.CaptureException(err)
	})
	hub.Flush(2 * time.Second)
}
