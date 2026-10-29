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
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	runnerlogsfs "github.com/superplanehq/superplane/pkg/runners/logs/fs"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"go.opentelemetry.io/otel"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestRunnerWebSocketDeliversAndCompletesTask(t *testing.T) {
	for _, scenario := range []string{"complete", "cancel", "queued subscription"} {
		t.Run(scenario, func(t *testing.T) {
			cancelOnHeartbeat := scenario == "cancel"
			resource := support.Setup(t)
			signer := jwt.NewSigner("registration-secret")
			runner, registration, fleet := createRegistrationFixture(t)
			db := database.DB(t.Context())

			taskID := uuid.New()
			var subscription *models.FactoryCodexConnection
			document := map[string]any{
				"id":             taskID.String(),
				"run_mode":       "bash_script",
				"script":         "echo hello",
				"execution_mode": "host",
			}
			if scenario == "queued subscription" {
				require.NoError(t, models.EnableExperimentalFeature(resource.Organization.ID, features.FeatureCodexSubscriptions))
				factory, err := models.CreateFactory(db, resource.Organization.ID, support.RandomName("factory"), "", "")
				require.NoError(t, err)
				subscription, err = models.CreateFactoryCodexConnection(db, resource.Organization.ID, factory.ID, resource.User, time.Now().Add(-time.Hour))
				require.NoError(t, err)
				require.NoError(t, db.Model(subscription).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
				subscription, err = models.ReserveFactoryCodexConnection(db, resource.Organization.ID, factory.ID, support.CreateCodexOwnedRun(t, db, subscription, taskID), taskID, time.Now().Add(-time.Hour), resource.Encryptor)
				require.NoError(t, err)
				expired, err := signer.GenerateWithClaims(-time.Minute, map[string]string{"purpose": runneraction.CodexSubscriptionTokenPurpose})
				require.NoError(t, err)
				document["environment"] = []runneraction.TaskEnvironmentVariable{{Name: runneraction.EnvCodexSubscriptionToken, Value: expired}}
				document["execution_timeout_seconds"] = 30
			}
			payload, err := json.Marshal(document)
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
			server, err := NewServer(signer, crypto.NewNoOpEncryptor(), runnerlogs.StoreFS, nil)
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
			if subscription != nil {
				var received struct {
					Environment []runneraction.TaskEnvironmentVariable `json:"environment"`
				}
				require.NoError(t, json.Unmarshal(delivered.Task, &received))
				require.Len(t, received.Environment, 1)
				scope, err := runneraction.ParseCodexSubscriptionToken(signer, received.Environment[0].Value)
				require.NoError(t, err)
				assert.Equal(t, task.ID, scope.TaskID)
				assert.Equal(t, subscription.ID, scope.ConnectionID)
				claims, err := signer.ValidateAndGetClaims(received.Environment[0].Value)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, int64(claims["exp"].(float64)), time.Now().Add(29*time.Second).Unix())
			}

			if cancelOnHeartbeat {
				require.NoError(t, task.RequestCancel(db, time.Now()))
				require.NoError(t, socket.WriteJSON(typedMessage{Type: messageTypePong}))
				require.NoError(t, socket.SetReadDeadline(time.Now().Add(2*time.Second)))
				var cancel taskControlMessage
				require.NoError(t, socket.ReadJSON(&cancel))
				assert.Equal(t, messageTypeCancel, cancel.Type)
				assert.Equal(t, task.ID.String(), cancel.TaskID)
			}

			requestID := uuid.New()
			require.NoError(t, socket.WriteJSON(completeMessage{
				Type:      messageTypeComplete,
				RequestID: requestID.String(),
				TaskID:    task.ID.String(),
				ExitCode:  0,
				Canceled:  cancelOnHeartbeat,
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
			expectedState := models.RunnerTaskStateSucceeded
			if cancelOnHeartbeat {
				expectedState = models.RunnerTaskStateCanceled
			}
			assert.Equal(t, expectedState, reloadedTask.State)
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

		})
	}
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
	execution, poll := createRunnerTaskPoll(t, resource, task)

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
	server, err := NewServer(signer, resource.Encryptor, runnerlogs.StoreFS, workers.NewNodeRequestExecutor(resource.Encryptor, resource.Registry, "", resource.AuthService))
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
	require.NoError(t, db.First(execution, "id = ?", execution.ID).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, execution.State, "execution must finish before completion is acknowledged")
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, execution.Result)
	require.NoError(t, db.First(poll, "id = ?", poll.ID).Error)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, poll.State)

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

func TestCompleteRunnerTaskRetriesPendingRequest(t *testing.T) {
	for _, name := range []string{"hook failure", "request locked"} {
		t.Run(name, func(t *testing.T) {
			resource := support.Setup(t)
			defer resource.Close()
			db := database.DB(t.Context())
			runner, task, fleet, connectionID := createRunningRunnerTask(t, resource)
			execution, request := createRunnerTaskPoll(t, resource, task)
			// Completion must use the component's pending request without depending
			// on its action name or parameter schema.
			componentName := "taskCompletionTest"
			resource.Registry.Actions[componentName] = impl.NewDummyAction(impl.DummyActionOptions{
				Name:  componentName,
				Hooks: []core.Hook{{Name: "completeTask", Type: core.HookTypeInternal}},
				HandleHookFunc: func(ctx core.ActionHookContext) error {
					return ctx.ExecutionState.Emit("passed", "task.finished", []any{map[string]any{"value": "ok"}})
				},
			})
			node, err := models.FindCanvasNode(db, execution.WorkflowID, execution.NodeID)
			require.NoError(t, err)
			require.NoError(t, db.Model(node).Update("ref", datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: componentName},
			})).Error)
			require.NoError(t, db.Model(request).Update("spec", datatypes.NewJSONType(models.NodeExecutionRequestSpec{
				InvokeAction: &models.InvokeAction{ActionName: "completeTask", Parameters: map[string]any{}},
			})).Error)

			if name == "hook failure" {
				node, err := models.FindCanvasNode(db, execution.WorkflowID, execution.NodeID)
				require.NoError(t, err)
				require.NoError(t, db.Model(node).Update("ref", datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "missing"},
				})).Error)
			}
			releaseLock := func() {}
			if name == "request locked" {
				tx := db.Begin()
				require.NoError(t, tx.Error)
				defer tx.Rollback()
				_, err := models.LockNodeRequestNow(tx, request.ID)
				require.NoError(t, err)
				releaseLock = func() { require.NoError(t, tx.Rollback().Error) }
			}
			executor := workers.NewNodeRequestExecutor(resource.Encryptor, resource.Registry, "", resource.AuthService)
			server, err := NewServer(jwt.NewSigner("secret"), resource.Encryptor, runnerlogs.StoreFS, executor)
			require.NoError(t, err)
			completion := completeMessage{RequestID: uuid.NewString(), TaskID: task.ID.String(), Result: json.RawMessage(`{"value":"ok"}`)}
			require.NoError(t, server.completeRunnerTask(t.Context(), runner, connectionID, completion, fleet.Slug, fleet.ScopeType))
			reloadedTask, err := models.FindRunnerTask(db, task.ID)
			require.NoError(t, err)
			assert.Equal(t, models.RunnerTaskStateSucceeded, reloadedTask.State)
			require.NoError(t, db.First(request, "id = ?", request.ID).Error)
			assert.Equal(t, models.NodeExecutionRequestStatePending, request.State, "request must remain available after a failed or skipped attempt")

			releaseLock()
			node, err = models.FindCanvasNode(db, execution.WorkflowID, execution.NodeID)
			require.NoError(t, err)
			require.NoError(t, db.Model(node).Update("ref", datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: componentName},
			})).Error)
			for range 2 {
				completion.RequestID = uuid.NewString()
				require.NoError(t, server.completeRunnerTask(t.Context(), runner, connectionID, completion, fleet.Slug, fleet.ScopeType))
			}
			require.NoError(t, db.First(execution, "id = ?", execution.ID).Error)
			assert.Equal(t, models.CanvasNodeExecutionStateFinished, execution.State)
			events, err := models.FindCanvasEventsForExecutions(db, []string{execution.ID.String()})
			require.NoError(t, err)
			require.Len(t, events, 1, "repeated completion must not emit duplicate outputs")
			assert.Equal(t, "passed", events[0].Channel)
		})
	}
}

func TestCompleteRunnerTaskDoesNotWaitForSlowExecutionRequest(t *testing.T) {
	resource := support.Setup(t)
	defer resource.Close()
	db := database.DB(t.Context())
	runner, task, fleet, connectionID := createRunningRunnerTask(t, resource)
	execution, request := createRunnerTaskPoll(t, resource, task)

	started := make(chan struct{})
	release := make(chan struct{})
	componentName := "slowTaskCompletionTest"
	resource.Registry.Actions[componentName] = impl.NewDummyAction(impl.DummyActionOptions{
		Name:  componentName,
		Hooks: []core.Hook{{Name: "poll", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			if err := ctx.ExecutionState.Emit("passed", "task.finished", []any{map[string]any{"value": "ok"}}); err != nil {
				return err
			}
			close(started)
			<-release
			return nil
		},
	})
	node, err := models.FindCanvasNode(db, execution.WorkflowID, execution.NodeID)
	require.NoError(t, err)
	require.NoError(t, db.Model(node).Update("ref", datatypes.NewJSONType(models.NodeRef{
		Component: &models.ComponentRef{Name: componentName},
	})).Error)
	executor := workers.NewNodeRequestExecutor(resource.Encryptor, resource.Registry, "", resource.AuthService)
	server, err := NewServer(jwt.NewSigner("secret"), resource.Encryptor, runnerlogs.StoreFS, executor)
	require.NoError(t, err)
	completion := completeMessage{RequestID: uuid.NewString(), TaskID: task.ID.String(), Result: json.RawMessage(`{"value":"ok"}`)}
	completed := make(chan error, 1)
	defer func() {
		close(release)
		require.Eventually(t, func() bool {
			updated, err := models.FindCanvasNodeRequest(db, request.ID)
			return err == nil && updated.State == models.NodeExecutionRequestStateCompleted
		}, 5*time.Second, 10*time.Millisecond, "the slow request must finish after the runner is released")
	}()
	go func() {
		completed <- server.completeRunnerTask(t.Context(), runner, connectionID, completion, fleet.Slug, fleet.ScopeType)
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the execution hook did not start")
	}
	select {
	case err := <-completed:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("slow cleanup must not block the runner completion reply")
	}
	reloadedTask, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateSucceeded, reloadedTask.State)
	pending, err := models.FindCanvasNodeRequest(db, request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStatePending, pending.State, "slow cleanup must still hold the request until its transaction commits")
}

func createRunningRunnerTask(t *testing.T, resource *support.ResourceRegistry) (*models.Runner, *models.RunnerTask, *models.RunnerFleet, uuid.UUID) {
	t.Helper()
	db := database.DB(t.Context())
	runner, _, fleet := createRegistrationFixture(t)
	connectionID := uuid.New()
	require.NoError(t, db.Model(runner).Updates(map[string]any{
		"state": models.RunnerStateIdle, "current_connection_id": connectionID,
	}).Error)
	task := &models.RunnerTask{
		ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
		QueuedAt: time.Now(), PayloadCiphertext: []byte(`{}`),
	}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return task.Start(tx, runner, runnerlogs.StoreFS, time.Now())
	}))
	return runner, task, fleet, connectionID
}

func createRunnerTaskPoll(t *testing.T, resource *support.ResourceRegistry, task *models.RunnerTask) (*models.CanvasNodeExecution, *models.CanvasNodeRequest) {
	t.Helper()
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, resource.Organization.ID, resource.User, []models.CanvasNode{{
		NodeID: "runner", Type: models.NodeTypeComponent,
		Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerBash"}}),
	}}, nil)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "runner", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "runner", event.ID, event.ID)
	require.NoError(t, db.Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, "runner", execution.ID, "task_id", task.ID.String()))
	poll := models.CanvasNodeRequest{
		ID: uuid.New(), WorkflowID: canvas.ID, NodeID: "runner", ExecutionID: &execution.ID,
		State: models.NodeExecutionRequestStatePending, Type: models.NodeRequestTypeInvokeAction,
		RunAt: time.Now().Add(time.Hour),
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{InvokeAction: &models.InvokeAction{
			ActionName: "poll", Parameters: map[string]any{"task_id": task.ID.String(), "organization_id": resource.Organization.ID.String()},
		}}),
	}
	require.NoError(t, db.Create(&poll).Error)

	return execution, &poll
}
