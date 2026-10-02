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
	runnerlogspostgres "github.com/superplanehq/superplane/pkg/runners/logs/postgres"
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

	activeStore := runnerlogspostgres.New()
	require.NoError(t, activeStore.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		Database:      db,
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
	server, err := NewServer(signer, crypto.NewNoOpEncryptor(), runnerlogs.StorePostgres)
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

	reloadedTask, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateSucceeded, reloadedTask.State)
	assert.JSONEq(t, `{"value":"ok"}`, string(reloadedTask.Result))
	lifecycle, err := reloadedTask.FindLifecycle(db)
	require.NoError(t, err)
	assert.Equal(t, runnerlogs.StorePostgres, lifecycle.ActiveStore)
	assert.Equal(t, models.RunnerTaskLogStateArchivable, lifecycle.State)

	reloadedRunner, err := models.FindRunner(db, runner.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStateTerminated, reloadedRunner.State)
	assert.Equal(t, models.RunnerTerminationTaskCompleted, *reloadedRunner.TerminationReason)
	_, err = models.FindRunnerByAccessTokenHash(db, crypto.HashToken(registered.AccessToken))
	require.NoError(t, err, "credential must remain valid for completion acknowledgement retries")
}
