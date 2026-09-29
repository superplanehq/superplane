package public

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	"github.com/superplanehq/superplane/test/support"
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
	require.NoError(t, database.Conn().Create(&models.TaskLogUpload{
		TaskID:            task.ID,
		NextChunkSequence: 2,
		TotalBytes:        13,
		UpdatedAt:         time.Now(),
	}).Error)
	require.NoError(t, provider.Put(
		t.Context(),
		runnerlogs.ChunkKey(task.OrganizationID, task.ID, 0),
		bytes.NewBufferString("first\n"),
		blob.PutOptions{ContentType: "application/x-ndjson"},
	))
	require.NoError(t, provider.Put(
		t.Context(),
		runnerlogs.ChunkKey(task.OrganizationID, task.ID, 1),
		bytes.NewBufferString("second\n"),
		blob.PutOptions{ContentType: "application/x-ndjson"},
	))

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

	live := runnerTaskLogsGET(t, server, signer, resource, canvasID, executionID, -1)
	require.Equal(t, http.StatusOK, live.Code)
	assert.Equal(t, "1", live.Header().Get("X-Last-Chunk"))
	assert.Equal(t, "first\nsecond\n", live.Body.String())

	require.NoError(t, database.Conn().Where("task_id = ?", task.ID).
		Delete(&models.TaskLogUpload{}).
		Error)
	var compressed bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressed)
	_, err = gzipWriter.Write([]byte("final\n"))
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())
	require.NoError(t, provider.Put(
		t.Context(),
		runnerlogs.FinalKey(task.OrganizationID, task.ID),
		bytes.NewReader(compressed.Bytes()),
		blob.PutOptions{
			ContentType:     "application/x-ndjson",
			ContentEncoding: "gzip",
		},
	))

	final := runnerTaskLogsGET(t, server, signer, resource, canvasID, executionID, -1)
	require.Equal(t, http.StatusOK, final.Code)
	assert.Equal(t, "gzip", final.Header().Get("Content-Encoding"))
	gzipReader, err := gzip.NewReader(final.Body)
	require.NoError(t, err)
	content, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.NoError(t, gzipReader.Close())
	assert.Equal(t, "final\n", string(content))
}

func runnerTaskLogsGET(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	resource *support.ResourceRegistry,
	canvasID, executionID uuid.UUID,
	afterChunk int64,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/api/v1/canvases/%s/node-executions/%s/runner-logs?after_chunk=%d",
			canvasID,
			executionID,
			afterChunk,
		),
		nil,
	)
	request.Header.Set("x-organization-id", resource.Organization.ID.String())
	token, err := authentication.GenerateAccountToken(
		signer,
		resource.Account.ID.String(),
		time.Now(),
		time.Hour,
	)
	require.NoError(t, err)
	request.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	return response
}
