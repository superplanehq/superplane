package runnerapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	runnerlogsfs "github.com/superplanehq/superplane/pkg/runners/logs/fs"
	"github.com/superplanehq/superplane/test/support"
	"go.opentelemetry.io/otel"
)

func TestRunnerWebSocketDeliversAndCompletesTask(t *testing.T) {
	resource := support.Setup(t)
	signer := jwt.NewSigner("registration-secret")
	runner, registration, fleet := createRegistrationFixture(t)
	db := database.DB(t.Context())

	taskID := uuid.New()
	payload, err := json.Marshal(map[string]any{
		"id":             taskID.String(),
		"run_mode":       "bash_script",
		"script":         "echo hello",
		"execution_mode": "host",
	})
	require.NoError(t, err)
	task := &models.RunnerTask{
		ID:                taskID,
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: payload,
		QueuedAt:          time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))

	activeStore, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, activeStore.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		MeterProvider: otel.GetMeterProvider(),
	}))
	previousStore := runnerlogs.Current()
	runnerlogs.SetCurrent(activeStore)
	t.Cleanup(func() {
		require.NoError(t, activeStore.Delete(context.Background(), task.ID))
		runnerlogs.SetCurrent(previousStore)
	})

	registrationToken, err := MintRegistrationToken(
		signer,
		runner,
		registration,
		fleet.Slug,
		&task.ID,
	)
	require.NoError(t, err)
	server, err := NewServer(signer, crypto.NewNoOpEncryptor(), runnerlogs.StoreFS)
	require.NoError(t, err)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	registerResponse := executeRegistrationRequest(
		t,
		server,
		registrationToken,
		runner.RunnerVersion,
	)
	require.Equal(t, http.StatusOK, registerResponse.Code, registerResponse.Body.String())
	var registered registerRunnerResponse
	require.NoError(t, json.Unmarshal(registerResponse.Body.Bytes(), &registered))

	headers := http.Header{"Authorization": []string{"Bearer " + registered.AccessToken}}
	socketURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/runner/v1/connect"
	socket, response, err := websocket.DefaultDialer.Dial(socketURL, headers)
	require.NoError(t, err)
	if response != nil {
		defer response.Body.Close()
	}
	defer socket.Close()

	require.NoError(t, socket.WriteJSON(helloMessage{
		Type:     messageTypeHello,
		RunnerID: runner.ID.String(),
		FleetID:  fleet.Slug,
		Version:  runner.RunnerVersion,
	}))

	var delivered taskMessage
	require.NoError(t, socket.ReadJSON(&delivered))
	assert.Equal(t, messageTypeTask, delivered.Type)

	requestID := uuid.New()
	require.NoError(t, socket.WriteJSON(completeMessage{
		Type:      messageTypeComplete,
		RequestID: requestID.String(),
		TaskID:    task.ID.String(),
		ExitCode:  0,
		Result:    json.RawMessage(`{"value":"ok"}`),
	}))

	var ack ackMessage
	require.NoError(t, socket.ReadJSON(&ack))
	assert.Equal(t, messageTypeAck, ack.Type)
	assert.Equal(t, requestID.String(), ack.RequestID)
	var shutdown typedMessage
	require.NoError(t, socket.ReadJSON(&shutdown))
	assert.Equal(t, messageTypeShutdown, shutdown.Type)

	reloadedTask, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateSucceeded, reloadedTask.State)
	assert.JSONEq(t, `{"value":"ok"}`, string(reloadedTask.Result))
	lifecycle, err := reloadedTask.FindLifecycle(db)
	require.NoError(t, err)
	assert.Equal(t, runnerlogs.StoreFS, lifecycle.ActiveStore)
	assert.Equal(t, models.RunnerTaskLogStateArchivable, lifecycle.State)

	reloadedRunner, err := models.FindRunner(db, runner.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStateTerminated, reloadedRunner.State)
	assert.Equal(t, models.RunnerTerminationTaskCompleted, *reloadedRunner.TerminationReason)
	_, err = models.FindRunnerByAccessTokenHash(db, crypto.HashToken(registered.AccessToken))
	require.NoError(t, err, "credential must remain valid for completion acknowledgement retries")
}

func TestRunnerWebSocketShutdownRequestCancelsBusyEphemeralRunner(t *testing.T) {
	resource := support.Setup(t)
	signer := jwt.NewSigner("registration-secret")
	runner, registration, fleet := createRegistrationFixture(t)
	db := database.DB(t.Context())

	taskID := uuid.New()
	payload, err := json.Marshal(map[string]any{
		"id":             taskID.String(),
		"run_mode":       "bash_script",
		"script":         "sleep 30",
		"execution_mode": "host",
	})
	require.NoError(t, err)
	task := &models.RunnerTask{
		ID:                taskID,
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: payload,
		QueuedAt:          time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))

	activeStore, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, activeStore.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		MeterProvider: otel.GetMeterProvider(),
	}))
	previousStore := runnerlogs.Current()
	runnerlogs.SetCurrent(activeStore)
	t.Cleanup(func() {
		require.NoError(t, activeStore.Delete(context.Background(), task.ID))
		runnerlogs.SetCurrent(previousStore)
	})

	registrationToken, err := MintRegistrationToken(
		signer,
		runner,
		registration,
		fleet.Slug,
		&task.ID,
	)
	require.NoError(t, err)
	server, err := NewServer(signer, crypto.NewNoOpEncryptor(), runnerlogs.StoreFS)
	require.NoError(t, err)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	registerResponse := executeRegistrationRequest(t, server, registrationToken, runner.RunnerVersion)
	require.Equal(t, http.StatusOK, registerResponse.Code, registerResponse.Body.String())
	var registered registerRunnerResponse
	require.NoError(t, json.Unmarshal(registerResponse.Body.Bytes(), &registered))

	headers := http.Header{"Authorization": []string{"Bearer " + registered.AccessToken}}
	socketURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/runner/v1/connect"
	socket, response, err := websocket.DefaultDialer.Dial(socketURL, headers)
	require.NoError(t, err)
	if response != nil {
		defer response.Body.Close()
	}
	defer socket.Close()

	require.NoError(t, socket.WriteJSON(helloMessage{
		Type:     messageTypeHello,
		RunnerID: runner.ID.String(),
		FleetID:  fleet.Slug,
		Version:  runner.RunnerVersion,
	}))
	var delivered taskMessage
	require.NoError(t, socket.ReadJSON(&delivered))
	require.Equal(t, messageTypeTask, delivered.Type)

	shutdownRequestID := uuid.NewString()
	require.NoError(t, socket.WriteJSON(shutdownRequestMessage{
		Type:          messageTypeShutdownRequest,
		RequestID:     shutdownRequestID,
		Reason:        shutdownReasonSignal,
		CurrentTaskID: task.ID.String(),
	}))
	var cancel taskControlMessage
	require.NoError(t, socket.ReadJSON(&cancel))
	assert.Equal(t, messageTypeCancel, cancel.Type)
	assert.Equal(t, task.ID.String(), cancel.TaskID)
	assert.Equal(t, shutdownRequestID, cancel.RequestID)
	assert.Equal(t, shutdownReasonSignal, cancel.Reason)

	reloadedTask, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	require.NotNil(t, reloadedTask.CancelRequestedAt)

	completionRequestID := uuid.NewString()
	require.NoError(t, socket.WriteJSON(completeMessage{
		Type:      messageTypeComplete,
		RequestID: completionRequestID,
		TaskID:    task.ID.String(),
		ExitCode:  130,
		Canceled:  true,
	}))
	var ack ackMessage
	require.NoError(t, socket.ReadJSON(&ack))
	assert.Equal(t, completionRequestID, ack.RequestID)
	var shutdown typedMessage
	require.NoError(t, socket.ReadJSON(&shutdown))
	assert.Equal(t, messageTypeShutdown, shutdown.Type)

	reloadedTask, err = models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateCanceled, reloadedTask.State)
	reloadedRunner, err := models.FindRunner(db, runner.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStateTerminated, reloadedRunner.State)
}

func TestRunnerWebSocketShutdownRequestTerminatesIdleRunner(t *testing.T) {
	support.Setup(t)
	signer := jwt.NewSigner("registration-secret")
	runner, registration, fleet := createRegistrationFixture(t)
	registrationToken, err := MintRegistrationToken(
		signer,
		runner,
		registration,
		fleet.Slug,
		nil,
	)
	require.NoError(t, err)
	server, err := NewServer(signer, crypto.NewNoOpEncryptor(), runnerlogs.StoreFS)
	require.NoError(t, err)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	registerResponse := executeRegistrationRequest(t, server, registrationToken, runner.RunnerVersion)
	require.Equal(t, http.StatusOK, registerResponse.Code, registerResponse.Body.String())
	var registered registerRunnerResponse
	require.NoError(t, json.Unmarshal(registerResponse.Body.Bytes(), &registered))

	headers := http.Header{"Authorization": []string{"Bearer " + registered.AccessToken}}
	socketURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/runner/v1/connect"
	socket, response, err := websocket.DefaultDialer.Dial(socketURL, headers)
	require.NoError(t, err)
	if response != nil {
		defer response.Body.Close()
	}
	defer socket.Close()

	require.NoError(t, socket.WriteJSON(helloMessage{
		Type:     messageTypeHello,
		RunnerID: runner.ID.String(),
		FleetID:  fleet.Slug,
		Version:  runner.RunnerVersion,
	}))
	require.NoError(t, socket.WriteJSON(shutdownRequestMessage{
		Type:      messageTypeShutdownRequest,
		RequestID: uuid.NewString(),
		Reason:    shutdownReasonSignal,
	}))
	var shutdown typedMessage
	require.NoError(t, socket.ReadJSON(&shutdown))
	assert.Equal(t, messageTypeShutdown, shutdown.Type)

	reloadedRunner, err := models.FindRunner(database.DB(t.Context()), runner.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStateTerminated, reloadedRunner.State)
	assert.Equal(t, models.RunnerTerminationInterrupted, *reloadedRunner.TerminationReason)
}
