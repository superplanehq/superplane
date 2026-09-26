package public

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

// A webhook can finalize an execution from inside the handler (e.g. the runner
// completing via a broker callback). The handler must broadcast the resulting
// execution state change, otherwise the node stays stuck "running" in the UI.
func Test__HandleWebhook_PublishesExecutionStateForFinalizedExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	const actionName = "dummy-webhook-action"
	const taskID = "task-123"

	// Usage metering (publishRunnerUsage) silently skips when the webhook
	// execution context has no organization ID, so capture it for assertion.
	var seenOrganizationID string
	r.Registry.Actions[actionName] = impl.NewDummyAction(impl.DummyActionOptions{
		Name: actionName,
		HandleWebhookFunc: func(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
			execCtx, err := ctx.FindExecutionByKV("task_id", taskID)
			if err != nil {
				return http.StatusNotFound, nil, nil
			}

			seenOrganizationID = execCtx.OrganizationID

			if err := execCtx.ExecutionState.Pass(); err != nil {
				return http.StatusInternalServerError, nil, err
			}

			return http.StatusOK, nil, nil
		},
	})

	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
		support.NewOIDCProvider(),
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService, false,
	)
	require.NoError(t, err)

	webhookID := uuid.New()
	require.NoError(t, database.Conn().Create(&models.Webhook{
		ID:     webhookID,
		State:  models.WebhookStateReady,
		Secret: []byte("secret"),
	}).Error)

	nodeID := "action-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: nodeID,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: actionName}}),
			},
		},
		[]models.Edge{},
	)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ?", canvas.ID).
		Where("node_id = ?", nodeID).
		Update("webhook_id", webhookID).
		Error)

	rootEvent := &models.CanvasEvent{
		WorkflowID: canvas.ID,
		NodeID:     nodeID,
		Channel:    "default",
		Data:       models.JSONValue{},
		State:      models.CanvasEventStatePending,
	}
	require.NoError(t, database.Conn().Create(rootEvent).Error)

	execution := &models.CanvasNodeExecution{
		WorkflowID:  canvas.ID,
		NodeID:      nodeID,
		RootEventID: rootEvent.ID,
		EventID:     rootEvent.ID,
		State:       models.CanvasNodeExecutionStateStarted,
	}
	require.NoError(t, database.Conn().Create(execution).Error)

	tx := database.Conn()
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(tx, canvas.ID, nodeID, execution.ID, "task_id", taskID))

	amqpURL, _ := config.RabbitMQURL()
	finishedConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionFinishedRoutingKey)
	finishedConsumer.Start()
	defer finishedConsumer.Stop()

	response := execRequest(server, requestParams{
		method: "POST",
		path:   "/webhooks/" + webhookID.String(),
		body:   []byte(`{"ok": true}`),
	})
	require.Equal(t, http.StatusOK, response.Code)

	// The webhook execution context carries the organization ID (runner usage
	// metering depends on it).
	assert.Equal(t, r.Organization.ID.String(), seenOrganizationID)

	// The execution is finished in the DB...
	updated, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updated.State)

	// ...and the execution.finished event is broadcast so the UI updates without a reload.
	assert.True(t, finishedConsumer.HasReceivedMessage())
}

func Test__HandleWebhook_PausedIntakeIgnoresLiveEvents(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	const triggerName = "dummy-paused-jira-intake"

	handleCount := 0
	r.Registry.Triggers[triggerName] = impl.NewDummyTrigger(impl.DummyTriggerOptions{
		Name: triggerName,
		HandleWebhookFunc: func(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
			handleCount++
			if err := ctx.Events.Emit("jira.issue", map[string]any{"action": "created"}); err != nil {
				return http.StatusInternalServerError, nil, err
			}
			return http.StatusOK, nil, nil
		},
	})

	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
		support.NewOIDCProvider(),
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService, false,
	)
	require.NoError(t, err)

	webhookID := uuid.New()
	require.NoError(t, database.Conn().Create(&models.Webhook{
		ID:     webhookID,
		State:  models.WebhookStateReady,
		Secret: []byte("secret"),
	}).Error)

	nodeID := "trigger-1"
	canvas, nodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        nodeID,
				Name:          nodeID,
				Type:          models.NodeTypeTrigger,
				Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: triggerName}}),
				Configuration: datatypes.NewJSONType(map[string]any{}),
			},
		},
		nil,
	)
	require.Len(t, nodes, 1)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ?", canvas.ID).
		Where("node_id = ?", nodeID).
		Update("webhook_id", webhookID).
		Error)

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	intake, err := factory.CreateIntake(database.Conn(), canvas.ID, models.FactoryIntakeSourceJiraIssues)
	require.NoError(t, err)

	postWebhook := func() {
		response := execRequest(server, requestParams{
			method: "POST",
			path:   "/webhooks/" + webhookID.String(),
			body:   []byte(`{"webhookEvent":"jira:issue_created"}`),
		})
		require.Equal(t, http.StatusOK, response.Code)
	}

	postWebhook()
	assert.Equal(t, 1, handleCount)
	support.VerifyCanvasNodeEventsCount(t, canvas.ID, nodeID, 1)

	require.NoError(t, intake.SetPaused(database.Conn(), true))
	postWebhook()
	assert.Equal(t, 1, handleCount)
	support.VerifyCanvasNodeEventsCount(t, canvas.ID, nodeID, 1)

	require.NoError(t, intake.SetPaused(database.Conn(), false))
	postWebhook()
	assert.Equal(t, 2, handleCount)
	support.VerifyCanvasNodeEventsCount(t, canvas.ID, nodeID, 2)
}
