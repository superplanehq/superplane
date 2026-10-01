package public

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	logpostgres "github.com/superplanehq/superplane/pkg/runners/logs/postgres"
	"github.com/superplanehq/superplane/test/support"
	"go.opentelemetry.io/otel"
	"gorm.io/datatypes"
)

func TestHandleRunnerTaskLogsReadsLiveAndFinalLogs(t *testing.T) {
	resource := support.Setup(t)
	defer resource.Close()
	server, signer := mustRunnerLiveLogServer(t, resource)
	provider, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	previousProvider := blob.Current()
	blob.SetCurrent(provider)
	t.Cleanup(func() { blob.SetCurrent(previousProvider) })
	activeStore := logpostgres.New()
	require.NoError(t, activeStore.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		Database:      database.DB(t.Context()),
		MeterProvider: otel.GetMeterProvider(),
	}))
	previousActiveStore := runnerlogs.Current()
	runnerlogs.SetCurrent(activeStore)
	t.Cleanup(func() { runnerlogs.SetCurrent(previousActiveStore) })

	fleet := models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          "task-log-test",
		ScopeType:     models.RunnerFleetScopeInstallation,
		Enabled:       true,
		Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
		RunnerVersion: "0.1.0",
	}
	require.NoError(t, fleet.Create(database.Conn()))
	task := models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateRunning,
		PayloadCiphertext: []byte("{}"),
		QueuedAt:          time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	require.NoError(t, database.Conn().Create(&task).Error)
	require.NoError(t, database.Conn().Create(&models.RunnerTaskLogLifecycle{
		TaskID:      task.ID,
		ActiveStore: runnerlogs.StorePostgres,
		State:       models.RunnerTaskLogStateActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}).Error)
	_, err = activeStore.Append(t.Context(), task.ID, 0, []byte("first\n"))
	require.NoError(t, err)
	_, err = activeStore.Append(t.Context(), task.ID, 1, []byte("second\n"))
	require.NoError(t, err)

	canvasID, executionID := createCanvasWithComponentExecution(
		t,
		resource,
		runneraction.ComponentName,
		"runner-logs",
		map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: task.ID.String(),
			runneraction.ExecutionMetadataTaskBackend:  "integrated",
		},
	)

	live := runnerTaskLogsGET(t, server, signer, resource, canvasID, executionID, "")
	require.Equal(t, http.StatusOK, live.Code)
	assert.Equal(t, models.RunnerTaskLogStateActive, live.Header().Get(runnerlogs.HeaderState))
	assert.Equal(t, "2", live.Header().Get(runnerlogs.HeaderCursor))
	assert.Equal(t, "first\nsecond\n", live.Body.String())

	incremental := runnerTaskLogsGET(t, server, signer, resource, canvasID, executionID, "1")
	require.Equal(t, http.StatusOK, incremental.Code)
	assert.Equal(t, "second\n", incremental.Body.String())

	var compressed bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressed)
	_, err = gzipWriter.Write([]byte("final\n"))
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())
	installationID, err := models.GetInstallationID(database.Conn())
	require.NoError(t, err)
	finalKey := runnerlogs.FinalKey(installationID, task.OrganizationID, task.ID)
	require.NoError(t, provider.Put(
		t.Context(),
		finalKey,
		bytes.NewReader(compressed.Bytes()),
		blob.PutOptions{
			ContentType:     "application/x-ndjson",
			ContentEncoding: "gzip",
		},
	))
	finalCursor := "2"
	require.NoError(t, database.Conn().Model(&models.RunnerTaskLogLifecycle{}).
		Where("task_id = ?", task.ID).
		Updates(map[string]any{
			"state":            models.RunnerTaskLogStateArchived,
			"final_object_key": finalKey,
			"final_cursor":     finalCursor,
		}).
		Error)

	caughtUp := runnerTaskLogsGET(t, server, signer, resource, canvasID, executionID, finalCursor)
	require.Equal(t, http.StatusNoContent, caughtUp.Code)
	assert.Equal(t, models.RunnerTaskLogStateArchived, caughtUp.Header().Get(runnerlogs.HeaderState))

	require.NoError(t, activeStore.Delete(t.Context(), task.ID))

	stale := runnerTaskLogsGET(t, server, signer, resource, canvasID, executionID, "1")
	require.Equal(t, http.StatusConflict, stale.Code)
	assert.Equal(t, "true", stale.Header().Get(runnerlogs.HeaderReset))

	final := runnerTaskLogsGET(t, server, signer, resource, canvasID, executionID, "")
	require.Equal(t, http.StatusOK, final.Code)
	assert.Equal(t, "gzip", final.Header().Get("Content-Encoding"))
	assert.Equal(t, models.RunnerTaskLogStateArchived, final.Header().Get(runnerlogs.HeaderState))
	assert.Equal(t, finalCursor, final.Header().Get(runnerlogs.HeaderCursor))
	gzipReader, err := gzip.NewReader(final.Body)
	require.NoError(t, err)
	content, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.NoError(t, gzipReader.Close())
	assert.Equal(t, "final\n", string(content))
}
