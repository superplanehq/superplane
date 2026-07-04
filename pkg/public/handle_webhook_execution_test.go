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

	r.Registry.Actions[actionName] = impl.NewDummyAction(impl.DummyActionOptions{
		Name: actionName,
		HandleWebhookFunc: func(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
			execCtx, err := ctx.FindExecutionByKV("task_id", taskID)
			if err != nil {
				return http.StatusNotFound, nil, nil
			}

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
		r.GitProvider,
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService,
		nil,
		false,
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

	// The execution is finished in the DB...
	updated, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updated.State)

	// ...and the execution.finished event is broadcast so the UI updates without a reload.
	assert.True(t, finishedConsumer.HasReceivedMessage())
}
