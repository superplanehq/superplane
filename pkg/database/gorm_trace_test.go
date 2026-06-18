package database

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	gormlogger "gorm.io/gorm/logger"
)

func TestDBOperation(t *testing.T) {
	assert.Equal(t, "SELECT", dbOperation("  select * from users"))
	assert.Equal(t, "query", dbOperation("   "))
}

func TestDBSpanName(t *testing.T) {
	assert.Equal(t, "db.select users", dbSpanName("SELECT * FROM users WHERE id = $1"))
	assert.Equal(t, "db.insert workflow_runs", dbSpanName("INSERT INTO workflow_runs (id) VALUES ($1)"))
	assert.Equal(t, "db.update users", dbSpanName("UPDATE users SET name = $1"))
	assert.Equal(t, "db.delete workflow_events", dbSpanName("DELETE FROM workflow_events WHERE id = $1"))
	assert.Equal(t, "db.select", dbSpanName("SELECT 1"))
}

func TestTruncateSQL(t *testing.T) {
	short := "SELECT 1"
	assert.Equal(t, short, truncateSQL(short))

	long := strings.Repeat("x", 600)
	assert.Len(t, truncateSQL(long), 512+3)
	assert.True(t, strings.HasSuffix(truncateSQL(long), "..."))
}

func TestGormTimeoutLoggerTraceRecordsDatabaseSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
	})

	ctx, parent := provider.Tracer("test").Start(context.Background(), "parent")

	logger := newGormTimeoutLogger(gormlogger.Discard)
	begin := time.Now().Add(-10 * time.Millisecond)
	logger.Trace(ctx, begin, func() (string, int64) {
		sql, params := `SELECT * FROM "secrets" WHERE "secrets"."value" = $1`, []interface{}{"super-secret-token"}
		sql, params = logger.(*gormTimeoutLogger).ParamsFilter(ctx, sql, params...)
		return `SELECT * FROM "secrets" WHERE "secrets"."value" = '` + params[0].(string) + `'`, 1
	}, nil)
	parent.End()

	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)

	var dbSpan tracetest.SpanStub
	for _, span := range spans {
		if strings.HasPrefix(span.Name, "db.select") && strings.Contains(span.Name, "secrets") {
			dbSpan = span
			break
		}
	}
	require.NotEmpty(t, dbSpan.Name)

	statement := attributeValue(dbSpan.Attributes, "db.statement")
	require.NotEmpty(t, statement)
	assert.NotContains(t, statement, "super-secret-token")
	assert.Contains(t, statement, `'?'`)
}

func attributeValue(attrs []attribute.KeyValue, key string) string {
	for _, attr := range attrs {
		if string(attr.Key) == key {
			return attr.Value.AsString()
		}
	}
	return ""
}

func TestGormTimeoutLoggerTraceDelegatesWithoutRecordingSpan(t *testing.T) {
	base := &recordingLogger{}
	logger := newGormTimeoutLogger(base)

	logger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 0
	}, errors.New("boom"))

	assert.True(t, base.traceCalled)
}

type recordingLogger struct {
	traceCalled bool
}

func (l *recordingLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface { return l }
func (l *recordingLogger) Info(context.Context, string, ...interface{})           {}
func (l *recordingLogger) Warn(context.Context, string, ...interface{})           {}
func (l *recordingLogger) Error(context.Context, string, ...interface{})          {}
func (l *recordingLogger) Trace(context.Context, time.Time, func() (string, int64), error) {
	l.traceCalled = true
}
