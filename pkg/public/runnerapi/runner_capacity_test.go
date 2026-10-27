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
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	runnerlogsfs "github.com/superplanehq/superplane/pkg/runners/logs/fs"
	"github.com/superplanehq/superplane/test/support"
	"go.opentelemetry.io/otel"
)

func TestRunnerWebSocketWaitsForOrganizationCapacity(t *testing.T) {
	resource := support.Setup(t)
	t.Setenv(billingmodels.BillingEnabledEnv, "true")
	db := database.DB(t.Context())
	plan := &billingmodels.BillingPlan{
		Key: "runner_" + uuid.NewString()[:8], Name: "Runner capacity", Category: "paid", Visibility: "private",
		CreditFrequency: "none", CreditExpiration: "never", ParallelTaskLimit: new(1),
	}
	require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
	_, err := billingmodels.SetAdminOrganizationPlan(db, resource.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	runner, registration, fleet := createRegistrationFixture(t)
	otherFleet := *fleet
	otherFleet.ID = uuid.New()
	otherFleet.Slug = "other-amd64"
	require.NoError(t, otherFleet.Create(db))
	occupyingRunner := &models.Runner{
		ID: uuid.New(), FleetID: otherFleet.ID, State: models.RunnerStatePending,
		RunnerVersion: otherFleet.RunnerVersion, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, db.Create(occupyingRunner).Error)
	newTask := func(fleetID uuid.UUID) *models.RunnerTask {
		task := &models.RunnerTask{
			ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleetID,
			Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
			QueuedAt: time.Now(), PayloadCiphertext: []byte(`{"script":"echo queued"}`),
		}
		require.NoError(t, db.Create(task).Error)
		return task
	}
	occupyingTask := newTask(otherFleet.ID)
	require.NoError(t, occupyingTask.Reserve(db, occupyingRunner.ID))
	queued := newTask(fleet.ID)
	attachSession := func(task *models.RunnerTask, factoryModel *models.Factory) *models.FactoryPlanningSession {
		order, err := factoryModel.CreateWorkOrder(db, "Queued planning", "", &resource.User, nil, nil)
		require.NoError(t, err)
		canvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, resource, factoryModel.ID, "Backlog", "start")
		run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, entrypoint, models.CanvasRunStateStarted, "")
		require.NoError(t, err)
		session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
			Repository: "acme/payments", CanvasID: canvas.ID, CanvasRunID: run.ID, WorkOrderID: order.ID,
		})
		require.NoError(t, err)
		event := support.EmitCanvasEventForNode(t, canvas.ID, entrypoint, "default", nil)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, entrypoint, event.ID, event.ID)
		require.NoError(t, db.Model(execution).Update("run_id", run.ID).Error)
		require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, entrypoint, execution.ID, "task_id", task.ID.String()))
		return session
	}
	ownFactory, err := models.CreateFactory(db, resource.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	otherFactory, err := models.CreateFactory(db, resource.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	session := attachSession(queued, ownFactory)
	attachSession(newTask(otherFleet.ID), otherFactory)
	attachSession(newTask(otherFleet.ID), otherFactory)
	type boardUpdate struct {
		factoryID string
		orderID   string
		reason    string
		state     string
	}
	updates := make(chan boardUpdate, 12)
	restorePublisher := messages.SetFactoryWorkOrderUpdatedPublisherForTest(func(factoryID, orderID, reason string) error {
		// Read through another connection: the notification must follow commit.
		stored, findErr := models.FindRunnerTask(database.DB(t.Context()), queued.ID)
		if findErr != nil {
			return findErr
		}
		updates <- boardUpdate{factoryID: factoryID, orderID: orderID, reason: reason, state: stored.State}
		return nil
	})
	defer restorePublisher()
	assertUpdates := func(expected []boardUpdate) {
		t.Helper()
		var received []boardUpdate
		for range len(expected) {
			select {
			case update := <-updates:
				received = append(received, update)
			case <-time.After(time.Second):
				t.Fatal("the planning boards did not receive the runner capacity update")
			}
		}
		assert.ElementsMatch(t, expected, received)
		assert.Empty(t, updates, "reserved-to-running must not refresh other planning boards")
	}

	activeStore, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, activeStore.Setup(runnerlogs.SetupContext{
		Context: t.Context(), MeterProvider: otel.GetMeterProvider(),
	}))
	previousStore := runnerlogs.Current()
	runnerlogs.SetCurrent(activeStore)
	t.Cleanup(func() {
		require.NoError(t, activeStore.Delete(context.Background(), queued.ID))
		runnerlogs.SetCurrent(previousStore)
	})
	signer := jwt.NewSigner("registration-secret")
	registrationToken, err := MintRegistrationToken(signer, runner, registration, fleet.Slug, nil)
	require.NoError(t, err)
	server, err := NewServer(signer, crypto.NewNoOpEncryptor(), runnerlogs.StoreFS, nil)
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
	require.NoError(t, socket.SetReadDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, socket.WriteJSON(helloMessage{
		Type: messageTypeHello, RunnerID: runner.ID.String(), FleetID: fleet.Slug, Version: runner.RunnerVersion,
	}))

	// A protocol response proves the full organization left this idle connection open.
	require.NoError(t, socket.WriteJSON(typedMessage{Type: "invalid"}))
	var invalid protocolErrorMessage
	require.NoError(t, socket.ReadJSON(&invalid))
	require.Equal(t, "invalid_message", invalid.Code)
	stillQueued, err := models.FindRunnerTask(db, queued.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, stillQueued.State)
	assert.Nil(t, stillQueued.RunnerID)

	require.NoError(t, occupyingRunner.Terminate(db, models.RunnerTerminationProvisioningFailed))
	require.NoError(t, socket.WriteJSON(typedMessage{Type: messageTypePong}))
	var delivered taskMessage
	require.NoError(t, socket.ReadJSON(&delivered))
	require.Equal(t, messageTypeTask, delivered.Type)
	assert.JSONEq(t, string(queued.PayloadCiphertext), string(delivered.Task))
	require.Eventually(t, func() bool {
		started, err := models.FindRunnerTask(db, queued.ID)
		return err == nil && started.State == models.RunnerTaskStateRunning &&
			started.RunnerID != nil && *started.RunnerID == runner.ID
	}, time.Second, 10*time.Millisecond)
	assertUpdates([]boardUpdate{
		{ownFactory.ID.String(), session.DraftWorkOrderID.String(), "order.updated", models.RunnerTaskStateReserved},
		{otherFactory.ID.String(), "", "runner_capacity", models.RunnerTaskStateReserved},
		{ownFactory.ID.String(), session.DraftWorkOrderID.String(), "order.updated", models.RunnerTaskStateRunning},
	})

	require.NoError(t, socket.WriteJSON(typedMessage{Type: "invalid"}))
	require.NoError(t, socket.ReadJSON(&invalid))
	assert.Equal(t, "invalid_message", invalid.Code)
	requestID := uuid.NewString()
	require.NoError(t, socket.WriteJSON(completeMessage{
		Type: messageTypeComplete, RequestID: requestID, TaskID: queued.ID.String(), ExitCode: 0,
	}))
	var ack ackMessage
	require.NoError(t, socket.ReadJSON(&ack))
	assert.Equal(t, requestID, ack.RequestID)
	assertUpdates([]boardUpdate{
		{ownFactory.ID.String(), session.DraftWorkOrderID.String(), "order.updated", models.RunnerTaskStateSucceeded},
		{otherFactory.ID.String(), "", "runner_capacity", models.RunnerTaskStateSucceeded},
	})
}
