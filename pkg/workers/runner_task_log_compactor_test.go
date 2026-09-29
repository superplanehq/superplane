package workers

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestRunnerTaskLogCompactorFinalizesTaskLogs(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	provider, err := filesystem.New(t.TempDir())
	require.NoError(t, err)

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
	require.NoError(t, db.Create(&models.TaskLogUpload{
		TaskID:            task.ID,
		NextChunkSequence: 2,
		TotalBytes:        13,
		UpdatedAt:         time.Now(),
	}).Error)

	putLogChunk(t, provider, task, 0, "first\n")
	putLogChunk(t, provider, task, 1, "second\n")

	compactor := NewRunnerTaskLogCompactor(provider, time.Second, time.Minute)
	require.NoError(t, compactor.Process(context.Background()))

	var upload models.TaskLogUpload
	require.NoError(t, db.Where("task_id = ?", task.ID).First(&upload).Error)
	require.NotNil(t, upload.FinalizingAt)

	_, err = provider.Head(
		context.Background(),
		runnerlogs.FinalKey(task.OrganizationID, task.ID),
	)
	require.NoError(t, err)

	finalReader, err := provider.Get(
		context.Background(),
		runnerlogs.FinalKey(task.OrganizationID, task.ID),
	)
	require.NoError(t, err)
	gzipReader, err := gzip.NewReader(finalReader)
	require.NoError(t, err)
	content, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.NoError(t, gzipReader.Close())
	require.NoError(t, finalReader.Close())
	assert.Equal(t, "first\nsecond\n", string(content))

	cleanupCompactor := NewRunnerTaskLogCompactor(provider, time.Second, 0)
	require.NoError(t, cleanupCompactor.Process(context.Background()))
	assert.Error(t, db.Where("task_id = ?", task.ID).First(&upload).Error)
	for sequence := int64(0); sequence < 2; sequence++ {
		_, err := provider.Head(
			context.Background(),
			runnerlogs.ChunkKey(task.OrganizationID, task.ID, sequence),
		)
		assert.ErrorIs(t, err, blob.ErrNotFound)
	}
}

func TestRunnerTaskLogCompactorSerializesWorkers(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)

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
	require.NoError(t, db.Create(&models.TaskLogUpload{
		TaskID:            task.ID,
		NextChunkSequence: 1,
		TotalBytes:        6,
		UpdatedAt:         time.Now(),
	}).Error)
	putLogChunk(t, store, task, 0, "first\n")

	provider := newBlockingFinalPutProvider(
		store,
		runnerlogs.FinalKey(task.OrganizationID, task.ID),
	)
	t.Cleanup(func() {
		close(provider.release)
	})
	first := NewRunnerTaskLogCompactor(provider, time.Second, time.Minute)
	second := NewRunnerTaskLogCompactor(provider, time.Second, time.Minute)

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- first.Process(t.Context())
	}()
	select {
	case <-provider.started:
	case <-time.After(5 * time.Second):
		t.Fatal("first compactor did not start final upload")
	}

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

type blockingFinalPutProvider struct {
	blob.Provider
	finalKey  string
	started   chan struct{}
	release   chan struct{}
	finalPuts atomic.Int32
}

func newBlockingFinalPutProvider(
	provider blob.Provider,
	finalKey string,
) *blockingFinalPutProvider {
	return &blockingFinalPutProvider{
		Provider: provider,
		finalKey: finalKey,
		started:  make(chan struct{}),
		release:  make(chan struct{}, 1),
	}
}

func (p *blockingFinalPutProvider) Put(
	ctx context.Context,
	key string,
	reader io.Reader,
	options blob.PutOptions,
) error {
	if key == p.finalKey {
		if p.finalPuts.Add(1) == 1 {
			close(p.started)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-p.release:
			}
		}
	}
	return p.Provider.Put(ctx, key, reader, options)
}

func putLogChunk(
	t *testing.T,
	provider blob.Provider,
	task models.RunnerTask,
	sequence int64,
	content string,
) {
	t.Helper()
	require.NoError(t, provider.Put(
		context.Background(),
		runnerlogs.ChunkKey(task.OrganizationID, task.ID, sequence),
		bytes.NewBufferString(content),
		blob.PutOptions{ContentType: "application/x-ndjson"},
	))
}
