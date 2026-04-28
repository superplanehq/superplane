package database

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	gormlogger "gorm.io/gorm/logger"
)

// gormTimeoutLogger wraps GORM's logger and logs a clear line when PostgreSQL
// reports statement_timeout or idle_in_transaction_session_timeout on a query.
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

// postgresTimeoutKind is non-empty when err is a PostgreSQL session timeout.
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
	switch classifyPostgresSessionTimeout(err) {
	case postgresTimeoutStatement:
		log.Printf("[database] PostgreSQL statement_timeout exceeded: %v", err)
	case postgresTimeoutIdleInTransaction:
		log.Printf("[database] PostgreSQL idle_in_transaction_session_timeout exceeded: %v", err)
	}
}
