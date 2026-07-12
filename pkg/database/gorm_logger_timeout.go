package database

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
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

	parent := trace.SpanFromContext(ctx)
	if parent != nil && parent.IsRecording() {
		sql, rowsAffected := fc()
		end := time.Now()

		_, span := otel.Tracer("superplane").Start(
			ctx,
			dbSpanName(sql),
			trace.WithTimestamp(begin),
			trace.WithSpanKind(trace.SpanKindClient),
		)
		span.SetAttributes(
			semconv.DBSystemNamePostgreSQL,
			attribute.String("db.operation", dbOperation(sql)),
			attribute.String("db.statement", truncateSQL(sanitizeSQLStatement(sql))),
			attribute.Int64("db.rows_affected", rowsAffected),
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End(trace.WithTimestamp(end))
		return
	}

	w.base.Trace(ctx, begin, fc, err)
}

func (w *gormTimeoutLogger) ParamsFilter(
	ctx context.Context,
	sql string,
	params ...interface{},
) (string, []interface{}) {
	return sql, redactSQLParams(params...)
}

func dbOperation(sql string) string {
	fields := strings.Fields(strings.TrimSpace(sql))
	if len(fields) == 0 {
		return "query"
	}

	return strings.ToUpper(fields[0])
}

func dbSpanName(sql string) string {
	operation := dbOperation(sql)
	table := dbTableHint(sql)
	if table == "" {
		return "db." + strings.ToLower(operation)
	}

	return "db." + strings.ToLower(operation) + " " + table
}

func dbTableHint(sql string) string {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	switch {
	case strings.HasPrefix(upper, "SELECT"):
		if idx := strings.Index(upper, " FROM "); idx >= 0 {
			return firstIdentifier(upper[idx+6:])
		}
	case strings.HasPrefix(upper, "INSERT INTO "):
		return firstIdentifier(upper[len("INSERT INTO "):])
	case strings.HasPrefix(upper, "UPDATE "):
		return firstIdentifier(upper[len("UPDATE "):])
	case strings.HasPrefix(upper, "DELETE FROM "):
		return firstIdentifier(upper[len("DELETE FROM "):])
	}

	return ""
}

func firstIdentifier(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}

	end := strings.IndexAny(source, " \t\n(,;")
	if end == -1 {
		return strings.ToLower(source)
	}

	return strings.ToLower(strings.TrimSpace(source[:end]))
}

func truncateSQL(sql string) string {
	const maxLen = 512
	if len(sql) <= maxLen {
		return sql
	}

	return sql[:maxLen] + "..."
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
