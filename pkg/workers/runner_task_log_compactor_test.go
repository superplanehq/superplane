package workers

import (
	"compress/gzip"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	runnerlogsfs "github.com/superplanehq/superplane/pkg/runners/logs/fs"
	"github.com/superplanehq/superplane/test/support"
	"go.opentelemetry.io/otel"
	"gorm.io/datatypes"
)

func TestRunnerTaskLogCompactorFinalizesTaskLogs(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	provider, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	installationID, err := models.GetInstallationID(db)
	require.NoError(t, err)
	activeStore, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, activeStore.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		MeterProvider: otel.GetMeterProvider(),
	}))

	fleet := models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          "test-log-compaction",
		ScopeType:     models.RunnerFleetScopeInstallation,
		Enabled:       true,
		Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
		RunnerVersion: "0.1.0",
	}
	require.NoError(t, fleet.Create(db))

	task := models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateSucceeded,
		PayloadCiphertext: []byte("{}"),
		QueuedAt:          time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	require.NoError(t, db.Create(&task).Error)
	require.NoError(t, db.Create(&models.RunnerTaskLogLifecycle{
		TaskID:      task.ID,
		ActiveStore: runnerlogs.StoreFS,
		State:       models.RunnerTaskLogStateArchivable,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}).Error)

	require.NoError(t, activeStore.Initialize(t.Context(), task.ID))
	_, err = activeStore.Append(t.Context(), task.ID, 0, []byte("first\n"))
	require.NoError(t, err)
	_, err = activeStore.Append(t.Context(), task.ID, 1, []byte("second\n"))
	require.NoError(t, err)

	compactor := NewRunnerTaskLogCompactor(provider, activeStore, installationID, time.Second, time.Minute)
	require.NoError(t, compactor.Process(context.Background()))

	var lifecycle models.RunnerTaskLogLifecycle
	require.NoError(t, db.Where("task_id = ?", task.ID).First(&lifecycle).Error)
	assert.Equal(t, models.RunnerTaskLogStateArchived, lifecycle.State)
	require.NotNil(t, lifecycle.FinalCursor)
	require.NotNil(t, lifecycle.CleanupAfter)

	_, err = provider.Head(
		context.Background(),
		runnerlogs.FinalKey(installationID, task.OrganizationID, task.ID),
	)
	require.NoError(t, err)

	finalReader, err := provider.Get(
		context.Background(),
		runnerlogs.FinalKey(installationID, task.OrganizationID, task.ID),
	)
	require.NoError(t, err)
	gzipReader, err := gzip.NewReader(finalReader)
	require.NoError(t, err)
	content, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.NoError(t, gzipReader.Close())
	require.NoError(t, finalReader.Close())
	assert.Equal(t, "first\nsecond\n", string(content))

	require.NoError(t, db.Model(&models.RunnerTaskLogLifecycle{}).
		Where("task_id = ?", task.ID).
		Update("cleanup_after", time.Now().Add(-time.Second)).
		Error)
	cleanupCompactor := NewRunnerTaskLogCompactor(provider, activeStore, installationID, time.Second, 0)
	require.NoError(t, cleanupCompactor.Process(context.Background()))
	var cleanedLifecycle models.RunnerTaskLogLifecycle
	require.NoError(t, db.Where("task_id = ?", task.ID).First(&cleanedLifecycle).Error)
	assert.Equal(t, models.RunnerTaskLogStateArchived, cleanedLifecycle.State)
	assert.Nil(t, cleanedLifecycle.CleanupAfter)
	_, err = activeStore.ReadAfter(t.Context(), task.ID, "")
	assert.ErrorIs(t, err, runnerlogs.ErrNotFound)
}

func TestRunnerTaskLogCompactorMeasuresUncompressedSize(t *testing.T) {
	provider, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	compactor := NewRunnerTaskLogCompactor(provider, nil, "test", time.Second, time.Minute)

	content := strings.Repeat("line\n", 200)
	size, err := compactor.writeFinalObject(t.Context(), "test-log", strings.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)

	object, err := provider.Head(t.Context(), "test-log")
	require.NoError(t, err)
	assert.Less(t, object.Size, size)
}

func TestRunnerTaskLogCompactorArchivesTaskWithoutChunks(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	provider, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	installationID, err := models.GetInstallationID(db)
	require.NoError(t, err)
	activeStore, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, activeStore.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		MeterProvider: otel.GetMeterProvider(),
	}))

	fleet := models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          "test-empty-log-compaction",
		ScopeType:     models.RunnerFleetScopeInstallation,
		Enabled:       true,
		Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
		RunnerVersion: "0.1.0",
	}
	require.NoError(t, fleet.Create(db))

	task := models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateSucceeded,
		PayloadCiphertext: []byte("{}"),
		QueuedAt:          time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	require.NoError(t, db.Create(&task).Error)
	require.NoError(t, db.Create(&models.RunnerTaskLogLifecycle{
		TaskID:      task.ID,
		ActiveStore: runnerlogs.StoreFS,
		State:       models.RunnerTaskLogStateArchivable,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}).Error)

	require.NoError(t, activeStore.Initialize(t.Context(), task.ID))
	compactor := NewRunnerTaskLogCompactor(
		provider,
		activeStore,
		installationID,
		time.Second,
		time.Minute,
	)
	require.NoError(t, compactor.Process(t.Context()))

	var lifecycle models.RunnerTaskLogLifecycle
	require.NoError(t, db.Where("task_id = ?", task.ID).First(&lifecycle).Error)
	assert.Equal(t, models.RunnerTaskLogStateArchived, lifecycle.State)
	require.NotNil(t, lifecycle.FinalCursor)
	assert.Equal(t, "0", *lifecycle.FinalCursor)

	finalReader, err := provider.Get(
		t.Context(),
		runnerlogs.FinalKey(installationID, task.OrganizationID, task.ID),
	)
	require.NoError(t, err)
	gzipReader, err := gzip.NewReader(finalReader)
	require.NoError(t, err)
	content, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.NoError(t, gzipReader.Close())
	require.NoError(t, finalReader.Close())
	assert.Empty(t, content)
}

func TestRunnerTaskLogCompactorRejectsMissingActiveLogs(t *testing.T) {
	provider, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	activeStore, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, activeStore.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		MeterProvider: otel.GetMeterProvider(),
	}))

	compactor := NewRunnerTaskLogCompactor(
		provider,
		activeStore,
		uuid.NewString(),
		time.Second,
		time.Minute,
	)
	err = compactor.processTask(t.Context(), LogArchivingCandidate{
		TaskID:          uuid.New(),
		OrganizationID:  uuid.New(),
		ActiveStore:     runnerlogs.StoreFS,
		State:           models.RunnerTaskLogStateArchiving,
		ProcessingUntil: time.Now().Add(time.Minute),
	})

	require.ErrorIs(t, err, runnerlogs.ErrNotFound)
}

func TestRunnerTaskLogCompactorSerializesWorkers(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	installationID, err := models.GetInstallationID(db)
	require.NoError(t, err)
	activeStore, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, activeStore.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		MeterProvider: otel.GetMeterProvider(),
	}))

	fleet := models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          "test-concurrent-log-compaction",
		ScopeType:     models.RunnerFleetScopeInstallation,
		Enabled:       true,
		Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
		RunnerVersion: "0.1.0",
	}
	require.NoError(t, fleet.Create(db))

	task := models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateSucceeded,
		PayloadCiphertext: []byte("{}"),
		QueuedAt:          time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	require.NoError(t, db.Create(&task).Error)
	require.NoError(t, db.Create(&models.RunnerTaskLogLifecycle{
		TaskID:      task.ID,
		ActiveStore: runnerlogs.StoreFS,
		State:       models.RunnerTaskLogStateArchivable,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}).Error)
	require.NoError(t, activeStore.Initialize(t.Context(), task.ID))
	_, err = activeStore.Append(t.Context(), task.ID, 0, []byte("first\n"))
	require.NoError(t, err)

	provider := newBlockingFinalPutProvider(
		store,
		runnerlogs.FinalKey(installationID, task.OrganizationID, task.ID),
	)
	t.Cleanup(func() {
		close(provider.release)
	})
	first := NewRunnerTaskLogCompactor(provider, activeStore, installationID, time.Second, time.Minute)
	second := NewRunnerTaskLogCompactor(provider, activeStore, installationID, time.Second, time.Minute)

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- first.Process(t.Context())
	}()
	select {
	case <-provider.started:
	case <-time.After(5 * time.Second):
		t.Fatal("first compactor did not start final upload")
	}
	var lifecycle models.RunnerTaskLogLifecycle
	require.NoError(t, db.Where("task_id = ?", task.ID).First(&lifecycle).Error)
	assert.Equal(t, models.RunnerTaskLogStateArchiving, lifecycle.State)

	secondDone := make(chan error, 1)
	go func() {
		secondDone <- second.Process(t.Context())
	}()
	select {
	case err := <-secondDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("second compactor waited for the locked task")
	}
	assert.Equal(t, int32(1), provider.finalPuts.Load())

	provider.release <- struct{}{}
	require.NoError(t, <-firstDone)
	assert.Equal(t, int32(1), provider.finalPuts.Load())
}
