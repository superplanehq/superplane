package postgres

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestSetupIsConcurrentAndIdempotent(t *testing.T) {
	store := newTestStore(t)
	setupContext := testSetupContext(t)

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.Setup(setupContext)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}

func TestSetupRequiresDependencies(t *testing.T) {
	store := New()

	err := store.Setup(runnerlogs.SetupContext{})
	require.EqualError(t, err, "PostgreSQL active log store requires a context")

	err = store.Setup(runnerlogs.SetupContext{Context: t.Context()})
	require.EqualError(t, err, "PostgreSQL active log store requires a database")

	err = store.Setup(runnerlogs.SetupContext{
		Context:  t.Context(),
		Database: database.DB(t.Context()),
	})
	require.EqualError(t, err, "PostgreSQL active log store requires a meter provider")
}

func TestOperationsRecordPostgresMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	store := New()
	require.NoError(t, store.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		Database:      database.DB(t.Context()),
		MeterProvider: provider,
	}))
	taskID := uuid.New()
	_, err := store.Append(t.Context(), taskID, 0, []byte("line\n"))
	require.NoError(t, err)
	readResult(t, store, taskID, "")
	require.NoError(t, store.Delete(t.Context(), taskID))

	var metrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &metrics))
	names := map[string]bool{}
	for _, scope := range metrics.ScopeMetrics {
		for _, metric := range scope.Metrics {
			names[metric.Name] = true
		}
	}
	assert.True(t, names["runner_active_log.postgres.operations.total"])
	assert.True(t, names["runner_active_log.postgres.operation.duration.seconds"])
}

func TestAppendReadAndDelete(t *testing.T) {
	store := newTestStore(t)
	taskID := uuid.New()
	t.Cleanup(func() { require.NoError(t, store.Delete(context.Background(), taskID)) })

	result, err := store.Append(t.Context(), taskID, 0, []byte("first\n"))
	require.NoError(t, err)
	assert.False(t, result.Truncated)

	result, err = store.Append(t.Context(), taskID, 0, []byte("duplicate\n"))
	require.NoError(t, err)
	assert.False(t, result.Truncated)

	_, err = store.Append(t.Context(), taskID, 2, []byte("future\n"))
	require.ErrorIs(t, err, runnerlogs.ErrSequenceConflict)

	_, err = store.Append(t.Context(), taskID, 1, []byte("second\n"))
	require.NoError(t, err)

	first := readResult(t, store, taskID, "")
	assert.Equal(t, "first\nsecond\n", first.content)
	assert.Equal(t, "2", first.cursor)

	incremental := readResult(t, store, taskID, "1")
	assert.Equal(t, "second\n", incremental.content)
	assert.Equal(t, "2", incremental.cursor)

	require.NoError(t, store.Delete(t.Context(), taskID))
	require.NoError(t, store.Delete(t.Context(), taskID))
	_, err = store.ReadAfter(t.Context(), taskID, "")
	require.ErrorIs(t, err, runnerlogs.ErrNotFound)
}

func TestAppendTruncatesAtRetainedLimit(t *testing.T) {
	store := newTestStore(t)
	taskID := uuid.New()
	t.Cleanup(func() { require.NoError(t, store.Delete(context.Background(), taskID)) })

	line := `{"type":"line","text":"` + strings.Repeat("x", 1024) + `"}` + "\n"
	content := []byte(strings.Repeat(line, int(runnerlogs.MaxRetainedBytes/int64(len(line)))+2))

	result, err := store.Append(t.Context(), taskID, 0, content)
	require.NoError(t, err)
	assert.True(t, result.Truncated)

	read := readResult(t, store, taskID, "")
	assert.LessOrEqual(t, int64(len(read.content)), runnerlogs.MaxRetainedBytes)
	assert.True(t, strings.HasSuffix(read.content, string(truncationRecord)))
	assert.True(t, read.truncated)

	result, err = store.Append(t.Context(), taskID, 0, content)
	require.NoError(t, err)
	assert.True(t, result.Truncated)
}

func TestDeleteExpired(t *testing.T) {
	store := newTestStore(t)
	expiredID := uuid.New()
	activeID := uuid.New()
	t.Cleanup(func() {
		_ = store.Delete(context.Background(), expiredID)
		_ = store.Delete(context.Background(), activeID)
	})

	_, err := store.Append(t.Context(), expiredID, 0, []byte("old\n"))
	require.NoError(t, err)
	_, err = store.Append(t.Context(), activeID, 0, []byte("new\n"))
	require.NoError(t, err)

	require.NoError(t, database.Conn().Model(&activeLog{}).
		Where("task_id = ?", expiredID).
		Update("updated_at", time.Now().Add(-8*24*time.Hour)).
		Error)

	deleted, err := store.DeleteExpired(t.Context(), time.Now().Add(-7*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	_, err = store.ReadAfter(t.Context(), expiredID, "")
	require.True(t, errors.Is(err, runnerlogs.ErrNotFound))
	assert.Equal(t, "new\n", readResult(t, store, activeID, "").content)
}

type readValue struct {
	content   string
	cursor    string
	truncated bool
}

func readResult(t *testing.T, store *Store, taskID uuid.UUID, cursor string) readValue {
	t.Helper()
	result, err := store.ReadAfter(t.Context(), taskID, cursor)
	require.NoError(t, err)
	content, err := io.ReadAll(result.Content)
	require.NoError(t, err)
	require.NoError(t, result.Content.Close())
	return readValue{
		content:   string(content),
		cursor:    result.Cursor,
		truncated: result.Truncated,
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	require.NoError(t, database.VerifyTestDatabase(database.Conn()))
	store := New()
	require.NoError(t, store.Setup(testSetupContext(t)))
	return store
}

func testSetupContext(t *testing.T) runnerlogs.SetupContext {
	t.Helper()
	return runnerlogs.SetupContext{
		Context:       t.Context(),
		Database:      database.DB(t.Context()),
		MeterProvider: otel.GetMeterProvider(),
	}
}
