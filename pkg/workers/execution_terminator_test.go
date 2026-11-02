package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func Test__ExecutionTerminator__CancelsActiveExecution(t *testing.T) {
	r := support.Setup(t)

	amqpURL, _ := config.RabbitMQURL()
	executionFinishedConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionFinishedRoutingKey)
	executionFinishedConsumer.Start()
	defer executionFinishedConsumer.Stop()

	terminator := NewExecutionTerminator(amqpURL, r.AuthService, r.Encryptor, r.Registry)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID)

	require.NoError(t, execution.RequestCancellation(database.DB(t.Context()), &r.User))
	require.NoError(t, terminator.LockAndCancelExecution(*execution))

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)
	assert.True(t, executionFinishedConsumer.HasReceivedMessage())
}

func Test__ExecutionTerminator__InvokesComponentCancel(t *testing.T) {
	cancelCalled := false
	componentName := "execution_terminator_cancel_" + uuid.New().String()

	r := support.Setup(t)
	r.Registry.Actions[componentName] = impl.NewDummyAction(impl.DummyActionOptions{
		Name: componentName,
		CancelFunc: func(ctx core.ExecutionContext) error {
			cancelCalled = true
			assert.NotNil(t, ctx.Usage)
			return nil
		},
	})

	terminator := NewExecutionTerminator("", r.AuthService, r.Encryptor, r.Registry)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: componentName},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID)
	require.NoError(t, database.Conn().Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)

	require.NoError(t, execution.RequestCancellation(database.DB(t.Context()), &r.User))
	require.NoError(t, terminator.LockAndCancelExecution(*execution))

	assert.True(t, cancelCalled, "component Cancel should be invoked")

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)
}

func Test__ExecutionTerminator__CancelsWaitForPullRequestChecks(t *testing.T) {
	r := support.Setup(t)

	terminator := NewExecutionTerminator("", r.AuthService, r.Encryptor, r.Registry)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "wait-pr-checks",
				Name:   "Wait For Pull Request Checks",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "github.waitForPullRequestChecks"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "wait-pr-checks", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "wait-pr-checks", rootEvent.ID, rootEvent.ID)
	require.NoError(t, database.Conn().Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)

	runAt := time.Now().Add(time.Minute)
	require.NoError(t, execution.CreateRequest(database.Conn(), models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{ActionName: "evaluate", Parameters: map[string]any{}},
	}, &runAt))

	require.NoError(t, execution.RequestCancellation(database.DB(t.Context()), &r.User))
	require.NoError(t, terminator.LockAndCancelExecution(*execution))

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)

	pending, err := models.CountPendingRequestsForExecutionsInTransaction(database.Conn(), []uuid.UUID{execution.ID})
	require.NoError(t, err)
	assert.Zero(t, pending)
}

func Test__ExecutionTerminator__CancelsWithFrozenComponentName(t *testing.T) {
	cancelOriginalCalled := false
	cancelCurrentCalled := false
	originalComponent := "execution_terminator_frozen_orig_" + uuid.New().String()
	currentComponent := "execution_terminator_frozen_curr_" + uuid.New().String()

	r := support.Setup(t)
	r.Registry.Actions[originalComponent] = impl.NewDummyAction(impl.DummyActionOptions{
		Name: originalComponent,
		CancelFunc: func(ctx core.ExecutionContext) error {
			cancelOriginalCalled = true
			return nil
		},
	})
	r.Registry.Actions[currentComponent] = impl.NewDummyAction(impl.DummyActionOptions{
		Name: currentComponent,
		CancelFunc: func(ctx core.ExecutionContext) error {
			cancelCurrentCalled = true
			return nil
		},
	})

	terminator := NewExecutionTerminator("", r.AuthService, r.Encryptor, r.Registry)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: currentComponent},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state": models.CanvasNodeExecutionStateStarted,
		"metadata": datatypes.NewJSONType(map[string]any{
			models.CanvasNodeExecutionFrozenComponentKey: originalComponent,
		}),
	}).Error)

	require.NoError(t, execution.RequestCancellation(database.DB(t.Context()), &r.User))
	require.NoError(t, terminator.LockAndCancelExecution(*execution))

	assert.True(t, cancelOriginalCalled, "original/frozen component Cancel should be invoked")
	assert.False(t, cancelCurrentCalled, "current component Cancel should not be invoked")

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)
}

func Test__ExecutionTerminator__CompletesCancellationWhenActionNotFound(t *testing.T) {
	r := support.Setup(t)

	terminator := NewExecutionTerminator("", r.AuthService, r.Encryptor, r.Registry)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "non-existent-action-" + uuid.New().String()},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID)
	require.NoError(t, database.Conn().Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)

	require.NoError(t, execution.RequestCancellation(database.DB(t.Context()), &r.User))
	require.NoError(t, terminator.LockAndCancelExecution(*execution))

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)
}
