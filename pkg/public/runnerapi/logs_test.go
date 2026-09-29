package runnerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	"github.com/superplanehq/superplane/test/support"
)

func TestUploadTaskLogChunk(t *testing.T) {
	resource := support.Setup(t)
	signer := jwt.NewSigner("registration-secret")
	runner, registration, fleet := createRegistrationFixture(t)
	db := database.DB(t.Context())

	task := &models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte("{}"),
		QueuedAt:          time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))

	token, err := MintRegistrationToken(signer, runner, registration, fleet.Slug, &task.ID)
	require.NoError(t, err)
	server, err := NewServer(signer, crypto.NewNoOpEncryptor())
	require.NoError(t, err)
	response := executeRegistrationRequest(t, server, token, runner.RunnerVersion)
	require.Equal(t, http.StatusOK, response.Code)
	var registrationResponse registerRunnerResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &registrationResponse))
	accessToken := registrationResponse.AccessToken

	provider, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	previousProvider := blob.Current()
	blob.SetCurrent(provider)
	t.Cleanup(func() { blob.SetCurrent(previousProvider) })

	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	chunkURL := httpServer.URL + "/runner/v1/tasks/" + task.ID.String() + "/logs/chunks/"

	assertUploadStatus(t, chunkURL+"1", accessToken, "out of order\n", http.StatusConflict)
	assertUploadStatus(t, chunkURL+"0", accessToken, "first\n", http.StatusNoContent)
	assertUploadStatus(t, chunkURL+"0", accessToken, "duplicate\n", http.StatusNoContent)

	var upload models.TaskLogUpload
	require.NoError(t, db.Where("task_id = ?", task.ID).First(&upload).Error)
	assert.Equal(t, int64(1), upload.NextChunkSequence)
	assert.Equal(t, int64(len("first\n")), upload.TotalBytes)

	reader, err := provider.Get(
		context.Background(),
		runnerlogs.ChunkKey(task.OrganizationID, task.ID, 0),
	)
	require.NoError(t, err)
	defer reader.Close()
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "first\n", string(content))

	for sequence := int64(1); sequence < 3; sequence++ {
		assertUploadStatus(
			t,
			chunkURL+strconv.FormatInt(sequence, 10),
			accessToken,
			"next\n",
			http.StatusNoContent,
		)
	}
}

func assertUploadStatus(t *testing.T, url, token, content string, expected int) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(content))
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/x-ndjson")
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, expected, response.StatusCode)
}
