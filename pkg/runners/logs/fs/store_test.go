package fs

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNewProviderUsesConfiguredPath(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "active-logs")
	fallback := filepath.Join(base, "old-active-logs")
	t.Setenv("RUNNER_ACTIVE_LOG_FS_PATH", root)
	t.Setenv("RUNNER_ACTIVE_LOG_FS_FALLBACK_PATHS", fallback)

	store, err := NewProvider()
	require.NoError(t, err)
	require.NoError(t, store.Setup(testSetupContext(t)))

	assert.Equal(t, runnerlogs.StoreFS, store.Name())
	assert.DirExists(t, root)
	assert.DirExists(t, fallback)
	assert.Equal(t, []string{root, fallback}, store.paths)
}

func TestSetupIsConcurrentAndIdempotent(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "active-logs"))
	require.NoError(t, err)
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
	store, err := New(t.TempDir())
	require.NoError(t, err)

	err = store.Setup(runnerlogs.SetupContext{})
	require.EqualError(t, err, "FS active log store requires a context")

	err = store.Setup(runnerlogs.SetupContext{Context: t.Context()})
	require.EqualError(t, err, "FS active log store requires a meter provider")
}

func TestNewRejectsDuplicatePaths(t *testing.T) {
	root := t.TempDir()

	_, err := New(root, filepath.Join(root, "."))

	require.EqualError(
		t,
		err,
		"runner active log FS path "+strconv.Quote(root)+" is configured more than once",
	)
}

func TestOperationsRecordFSMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	store, err := New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, store.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		MeterProvider: provider,
	}))
	taskID := uuid.New()
	require.NoError(t, store.Initialize(t.Context(), taskID))
	_, err = store.Append(t.Context(), taskID, 0, []byte("line\n"))
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
	assert.True(t, names["runner_active_log.fs.operations.total"])
	assert.True(t, names["runner_active_log.fs.operation.duration.seconds"])
}

func TestAppendReadAndDelete(t *testing.T) {
	store := newTestStore(t)
	taskID := uuid.New()
	require.NoError(t, store.Initialize(t.Context(), taskID))

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
	assert.Equal(t, "13", first.cursor)

	incremental := readResult(t, store, taskID, "6")
	assert.Equal(t, "second\n", incremental.content)
	assert.Equal(t, "13", incremental.cursor)

	_, err = store.ReadAfter(t.Context(), taskID, "14")
	require.ErrorIs(t, err, runnerlogs.ErrInvalidCursor)

	require.NoError(t, store.Delete(t.Context(), taskID))
	require.NoError(t, store.Delete(t.Context(), taskID))
	_, err = store.ReadAfter(t.Context(), taskID, "")
	require.ErrorIs(t, err, runnerlogs.ErrNotFound)
}

func TestReadReturnsCommittedSnapshot(t *testing.T) {
	store := newTestStore(t)
	taskID := uuid.New()
	require.NoError(t, store.Initialize(t.Context(), taskID))

	_, err := store.Append(t.Context(), taskID, 0, []byte("first\n"))
	require.NoError(t, err)
	result, err := store.ReadAfter(t.Context(), taskID, "")
	require.NoError(t, err)

	_, err = store.Append(t.Context(), taskID, 1, []byte("second\n"))
	require.NoError(t, err)

	content, err := io.ReadAll(result.Content)
	require.NoError(t, err)
	require.NoError(t, result.Content.Close())
	assert.Equal(t, "first\n", string(content))
	assert.Equal(t, "6", result.Cursor)
}

func TestAppendDiscardsDataNotCommittedByManifest(t *testing.T) {
	store := newTestStore(t)
	taskID := uuid.New()
	require.NoError(t, store.Initialize(t.Context(), taskID))

	_, err := store.Append(t.Context(), taskID, 0, []byte("first\n"))
	require.NoError(t, err)

	file, err := os.OpenFile(
		filepath.Join(store.taskDir(store.primaryPath, taskID), dataFilename),
		os.O_APPEND|os.O_WRONLY,
		0,
	)
	require.NoError(t, err)
	_, err = file.WriteString("uncommitted\n")
	require.NoError(t, err)
	require.NoError(t, file.Sync())
	require.NoError(t, file.Close())

	_, err = store.Append(t.Context(), taskID, 1, []byte("second\n"))
	require.NoError(t, err)
	assert.Equal(t, "first\nsecond\n", readResult(t, store, taskID, "").content)
}

func TestAppendStateSurvivesStoreRestart(t *testing.T) {
	root := t.TempDir()
	first := newTestStoreAt(t, root)
	taskID := uuid.New()
	require.NoError(t, first.Initialize(t.Context(), taskID))

	_, err := first.Append(t.Context(), taskID, 0, []byte("first\n"))
	require.NoError(t, err)

	restarted := newTestStoreAt(t, root)
	_, err = restarted.Append(t.Context(), taskID, 0, []byte("duplicate\n"))
	require.NoError(t, err)
	_, err = restarted.Append(t.Context(), taskID, 1, []byte("second\n"))
	require.NoError(t, err)

	assert.Equal(t, "first\nsecond\n", readResult(t, restarted, taskID, "").content)
}

func TestPrimaryAndFallbackPathsSupportStorageMigration(t *testing.T) {
	base := t.TempDir()
	oldPath := filepath.Join(base, "zonal")
	newPath := filepath.Join(base, "enterprise")
	oldStore := newTestStoreAt(t, oldPath)
	oldTaskID := uuid.New()
	require.NoError(t, oldStore.Initialize(t.Context(), oldTaskID))
	_, err := oldStore.Append(t.Context(), oldTaskID, 0, []byte("old\n"))
	require.NoError(t, err)

	migratingStore, err := New(newPath, oldPath)
	require.NoError(t, err)
	require.NoError(t, migratingStore.Setup(testSetupContext(t)))
	assert.Equal(t, "old\n", readResult(t, migratingStore, oldTaskID, "").content)
	_, err = migratingStore.Append(t.Context(), oldTaskID, 1, []byte("continued\n"))
	require.NoError(t, err)
	assert.Equal(t, "old\ncontinued\n", readResult(t, migratingStore, oldTaskID, "").content)

	newTaskID := uuid.New()
	require.NoError(t, migratingStore.Initialize(t.Context(), newTaskID))
	_, err = migratingStore.Append(t.Context(), newTaskID, 0, []byte("new\n"))
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(newPath, newTaskID.String(), manifestFilename))
	assert.NoFileExists(t, filepath.Join(oldPath, newTaskID.String(), manifestFilename))

	require.NoError(t, migratingStore.Delete(t.Context(), oldTaskID))
	assert.NoDirExists(t, filepath.Join(oldPath, oldTaskID.String()))
}

func TestOperationsRejectTaskInMultiplePaths(t *testing.T) {
	base := t.TempDir()
	primaryPath := filepath.Join(base, "primary")
	fallbackPath := filepath.Join(base, "fallback")
	primaryStore := newTestStoreAt(t, primaryPath)
	fallbackStore := newTestStoreAt(t, fallbackPath)
	taskID := uuid.New()
	require.NoError(t, primaryStore.Initialize(t.Context(), taskID))
	require.NoError(t, fallbackStore.Initialize(t.Context(), taskID))

	store, err := New(primaryPath, fallbackPath)
	require.NoError(t, err)
	require.NoError(t, store.Setup(testSetupContext(t)))

	_, err = store.ReadAfter(t.Context(), taskID, "")
	require.EqualError(
		t,
		err,
		"active runner log "+taskID.String()+" exists in multiple FS paths",
	)
}

func TestFileLockSerializesStoreInstances(t *testing.T) {
	root := t.TempDir()
	first := newTestStoreAt(t, root)
	second := newTestStoreAt(t, root)
	taskID := uuid.New()
	require.NoError(t, first.Initialize(t.Context(), taskID))

	start := make(chan struct{})
	errs := make(chan error, 2)
	for _, operation := range []struct {
		store *Store
		value string
	}{
		{store: first, value: "first\n"},
		{store: second, value: "second\n"},
	} {
		go func(store *Store, value string) {
			<-start
			_, err := store.Append(t.Context(), taskID, 0, []byte(value))
			errs <- err
		}(operation.store, operation.value)
	}
	close(start)
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)

	content := readResult(t, first, taskID, "").content
	assert.Contains(t, []string{"first\n", "second\n"}, content)
}

func TestAppendRequiresInitialization(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Append(t.Context(), uuid.New(), 0, []byte("line\n"))

	require.ErrorIs(t, err, runnerlogs.ErrNotFound)
}

func TestInitializeIsIdempotent(t *testing.T) {
	store := newTestStore(t)
	taskID := uuid.New()
	require.NoError(t, store.Initialize(t.Context(), taskID))
	_, err := store.Append(t.Context(), taskID, 0, []byte("line\n"))
	require.NoError(t, err)

	require.NoError(t, store.Initialize(t.Context(), taskID))

	assert.Equal(t, "line\n", readResult(t, store, taskID, "").content)
}

func TestAppendTruncatesAtRetainedLimit(t *testing.T) {
	store := newTestStore(t)
	taskID := uuid.New()
	require.NoError(t, store.Initialize(t.Context(), taskID))
	line := `{"type":"line","text":"` + strings.Repeat("x", 1024) + `"}` + "\n"
	content := []byte(strings.Repeat(line, int(runnerlogs.MaxRetainedBytes/int64(len(line)))+2))

	result, err := store.Append(t.Context(), taskID, 0, content)
	require.NoError(t, err)
	assert.True(t, result.Truncated)

	read := readResult(t, store, taskID, "")
	assert.LessOrEqual(t, int64(len(read.content)), runnerlogs.MaxRetainedBytes)
	assert.True(t, strings.HasSuffix(read.content, runnerlogs.TruncationRecord))
	assert.True(t, read.truncated)

	result, err = store.Append(t.Context(), taskID, 0, content)
	require.NoError(t, err)
	assert.True(t, result.Truncated)
}

func TestDeleteExpired(t *testing.T) {
	store := newTestStore(t)
	expiredID := uuid.New()
	activeID := uuid.New()
	require.NoError(t, store.Initialize(t.Context(), expiredID))
	require.NoError(t, store.Initialize(t.Context(), activeID))

	_, err := store.Append(t.Context(), expiredID, 0, []byte("old\n"))
	require.NoError(t, err)
	_, err = store.Append(t.Context(), activeID, 0, []byte("new\n"))
	require.NoError(t, err)

	expiredDir := store.taskDir(store.primaryPath, expiredID)
	expired, found, err := readManifest(expiredDir)
	require.NoError(t, err)
	require.True(t, found)
	expired.UpdatedAt = time.Now().Add(-8 * 24 * time.Hour)
	require.NoError(t, writeManifest(expiredDir, expired))

	deleted, err := store.DeleteExpired(t.Context(), time.Now().Add(-7*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	_, err = store.ReadAfter(t.Context(), expiredID, "")
	require.ErrorIs(t, err, runnerlogs.ErrNotFound)
	assert.Equal(t, "new\n", readResult(t, store, activeID, "").content)
}

func TestDeleteExpiredScansFallbackPaths(t *testing.T) {
	base := t.TempDir()
	primaryPath := filepath.Join(base, "primary")
	fallbackPath := filepath.Join(base, "fallback")
	fallbackStore := newTestStoreAt(t, fallbackPath)
	taskID := uuid.New()
	require.NoError(t, fallbackStore.Initialize(t.Context(), taskID))

	taskDir := fallbackStore.taskDir(fallbackPath, taskID)
	current, found, err := readManifest(taskDir)
	require.NoError(t, err)
	require.True(t, found)
	current.UpdatedAt = time.Now().Add(-8 * 24 * time.Hour)
	require.NoError(t, writeManifest(taskDir, current))

	store, err := New(primaryPath, fallbackPath)
	require.NoError(t, err)
	require.NoError(t, store.Setup(testSetupContext(t)))
	deleted, err := store.DeleteExpired(t.Context(), time.Now().Add(-7*24*time.Hour))
	require.NoError(t, err)

	assert.Equal(t, int64(1), deleted)
	assert.NoDirExists(t, taskDir)
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
	return newTestStoreAt(t, t.TempDir())
}

func newTestStoreAt(t *testing.T, root string) *Store {
	t.Helper()
	store, err := New(root)
	require.NoError(t, err)
	require.NoError(t, store.Setup(testSetupContext(t)))
	return store
}

func testSetupContext(t *testing.T) runnerlogs.SetupContext {
	t.Helper()
	return runnerlogs.SetupContext{
		Context:       context.Background(),
		MeterProvider: otel.GetMeterProvider(),
	}
}
