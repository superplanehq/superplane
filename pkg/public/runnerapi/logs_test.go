package runnerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerclientapi "github.com/superplanehq/superplane/pkg/runners/api"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	runnerlogsfs "github.com/superplanehq/superplane/pkg/runners/logs/fs"
	"github.com/superplanehq/superplane/test/support"
	"go.opentelemetry.io/otel"
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
	server, err := NewServer(signer, crypto.NewNoOpEncryptor(), runnerlogs.StoreFS)
	require.NoError(t, err)
	response := executeRegistrationRequest(t, server, token, runner.RunnerVersion)
	require.Equal(t, http.StatusOK, response.Code)
	var registrationResponse registerRunnerResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &registrationResponse))
	accessToken := registrationResponse.AccessToken
	runner, err = models.FindRunner(db, runner.ID)
	require.NoError(t, err)

	store, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, store.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		MeterProvider: otel.GetMeterProvider(),
	}))
	previousStore := runnerlogs.Current()
	runnerlogs.SetCurrent(store)
	t.Cleanup(func() {
		require.NoError(t, store.Delete(context.Background(), task.ID))
		runnerlogs.SetCurrent(previousStore)
	})
	require.NoError(t, store.Initialize(t.Context(), task.ID))
	require.NoError(t, task.Start(db, runner, runnerlogs.StoreFS, time.Now()))

	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	chunkURL := httpServer.URL + "/runner/v1/tasks/" + task.ID.String() + "/logs/chunks/"

	maxUploadBytes := runnerclientapi.DefaultLogUploadPolicy().MaxUploadBytes()
	oversized := strings.Repeat("x", int(maxUploadBytes)+1)
	assertUploadStatus(t, chunkURL+"0", accessToken, oversized, http.StatusRequestEntityTooLarge)
	assertUploadStatus(t, chunkURL+"1", accessToken, "out of order\n", http.StatusConflict)
	assertUploadStatus(t, chunkURL+"0", accessToken, "first\n", http.StatusNoContent)
	assertUploadStatus(t, chunkURL+"0", accessToken, "duplicate\n", http.StatusNoContent)

	read, err := store.ReadAfter(t.Context(), task.ID, "")
	require.NoError(t, err)
	defer read.Content.Close()
	content, err := io.ReadAll(read.Content)
	require.NoError(t, err)
	assert.Equal(t, "first\n", string(content))
	assert.Equal(t, "6", read.Cursor)

	for sequence := int64(1); sequence < 3; sequence++ {
		assertUploadStatus(
			t,
			chunkURL+strconv.FormatInt(sequence, 10),
			accessToken,
			"next\n",
			http.StatusNoContent,
		)
	}

	paddingSize := int(runnerlogs.MaxRetainedBytes) - 1024 - len("first\nnext\nnext\n")
	padding := strings.Repeat("x", paddingSize-1) + "\n"
	appendResult, err := store.Append(t.Context(), task.ID, 3, []byte(padding))
	require.NoError(t, err)
	require.False(t, appendResult.Truncated)

	truncated := uploadChunk(t, chunkURL+"4", accessToken, strings.Repeat("x", 2048)+"\n")
	require.Equal(t, http.StatusNoContent, truncated.StatusCode)
	assert.Equal(t, runnerclientapi.LogUploadActionStop, truncated.Header.Get(runnerclientapi.HeaderLogUploadAction))
	require.NoError(t, truncated.Body.Close())

	stopped := uploadChunk(t, chunkURL+"5", accessToken, "ignored\n")
	require.Equal(t, http.StatusNoContent, stopped.StatusCode)
	assert.Equal(t, runnerclientapi.LogUploadActionStop, stopped.Header.Get(runnerclientapi.HeaderLogUploadAction))
	require.NoError(t, stopped.Body.Close())
}

func assertUploadStatus(t *testing.T, url, token, content string, expected int) {
	t.Helper()
	response := uploadChunk(t, url, token, content)
	defer response.Body.Close()
	assert.Equal(t, expected, response.StatusCode)
	if expected == http.StatusNoContent {
		assert.Equal(t, runnerclientapi.LogUploadActionContinue, response.Header.Get(runnerclientapi.HeaderLogUploadAction))
		assert.Equal(t, "65536", response.Header.Get(runnerclientapi.HeaderLogTargetChunkBytes))
		assert.Equal(t, "2500", response.Header.Get(runnerclientapi.HeaderLogFlushMinimumMS))
		assert.Equal(t, "5000", response.Header.Get(runnerclientapi.HeaderLogFlushMaximumMS))
	}
}

func uploadChunk(t *testing.T, url, token, content string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(content))
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/x-ndjson")
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	return response
}
