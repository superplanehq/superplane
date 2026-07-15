package workers

import (
	"errors"
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
	"github.com/superplanehq/superplane/pkg/registry"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func Test__NodeRequestWorker_InvokeTriggerAction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a schedule trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking the emitEvent action on the schedule trigger.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it completes successfully.
	//
	err := worker.LockAndProcessRequest(request)
	require.NoError(t, err)

	//
	// Verify the request was marked as completed.
	//
	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_InvokeTriggerAction_DefersRunTitleResolutionUntilEventEmit(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
					"customName":   "Minute {{ root().data.calendar.minute }}",
				}),
			},
		},
		[]models.Edge{},
	)

	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	err := worker.LockAndProcessRequest(request)
	require.NoError(t, err)

	var event models.CanvasEvent
	err = database.Conn().
		Where("workflow_id = ?", canvas.ID).
		Where("node_id = ?", triggerNode).
		First(&event).Error
	require.NoError(t, err)
	require.NotNil(t, event.CustomName)
	assert.Regexp(t, `^Minute \d{2}$`, *event.CustomName)

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_InvokeNodeComponentActionWithoutExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     componentNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "non-existent-action",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	err := worker.LockAndProcessRequest(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook non-existent-action not found for action noop")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_PreventsConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a schedule trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking a trigger action.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Have two workers call LockAndProcessRequest concurrently on the same request.
	// LockAndProcessRequest uses a transaction with locking, so only one should actually process.
	//
	results := make(chan error, 2)

	//
	// Create two workers and have them try to process the request concurrently.
	//
	go func() {
		worker1 := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)
		results <- worker1.LockAndProcessRequest(request)
	}()

	go func() {
		worker2 := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)
		results <- worker2.LockAndProcessRequest(request)
	}()

	// Collect results - both should succeed (return nil)
	// because LockAndProcessRequest returns nil when it can't acquire the lock
	result1 := <-results
	result2 := <-results
	assert.NoError(t, result1)
	assert.NoError(t, result2)

	//
	// Verify the request was marked as completed.
	//
	var updatedRequest models.CanvasNodeRequest
	err := database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	//
	// Verify that exactly one workflow event was emitted (proving only one worker processed it).
	//
	eventCount, err := models.CountCanvasEvents(canvas.ID, triggerNode)
	require.NoError(t, err)
	assert.Equal(t, int64(1), eventCount, "Expected exactly 1 workflow event, but found %d", eventCount)

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_UnsupportedRequestType(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request with an unsupported type.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       "unsupported-type",
		Spec:       datatypes.NewJSONType(models.NodeExecutionRequestSpec{}),
		State:      models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := worker.LockAndProcessRequest(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported node execution request type")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_MissingInvokeActionSpec(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request without an InvokeAction spec.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec:       datatypes.NewJSONType(models.NodeExecutionRequestSpec{}), // Missing InvokeAction
		State:      models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := worker.LockAndProcessRequest(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec is not specified")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_NonExistentTrigger(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a trigger node that references a non-existent trigger.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "non-existent-trigger"}}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking a trigger action.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := worker.LockAndProcessRequest(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trigger non-existent-trigger not registered")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_NonExistentAction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a schedule trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking a non-existent action.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "non-existent-action",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := worker.LockAndProcessRequest(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook non-existent-action not found for trigger schedule")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_CompletesDeletedNodeRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	triggerNode := "trigger-1"
	canvas, canvasNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	require.NoError(t, database.Conn().Delete(&canvasNodes[0]).Error)

	requests, err := models.ListNodeRequests()
	require.NoError(t, err)

	found := false
	for _, req := range requests {
		if req.ID == request.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "Request for deleted node should be returned by ListNodeRequests")

	err = worker.LockAndProcessRequest(request)
	require.NoError(t, err)

	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	eventCount, err := models.CountCanvasEvents(canvas.ID, triggerNode)
	require.NoError(t, err)
	assert.Zero(t, eventCount)

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_CancelsExecutionForDeletedNodeRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	componentNode := "component-1"
	canvas, canvasNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state": models.CanvasNodeExecutionStateStarted,
	}).Error)

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "noop",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	require.NoError(t, database.Conn().Delete(&canvasNodes[0]).Error)

	err := worker.LockAndProcessRequest(request)
	require.NoError(t, err)

	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	var updatedExecution models.CanvasNodeExecution
	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)
}

func Test__NodeRequestWorker_CancelsExecutionForDeletedNodeWhenComponentCancelFails(t *testing.T) {
	cancelCalled := false
	componentName := "cancel_failing_" + uuid.New().String()

	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name: componentName,
		CancelFunc: func(ctx core.ExecutionContext) error {
			cancelCalled = true
			return errors.New("cancel failed")
		},
	}))

	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	componentNode := "component-1"
	canvas, canvasNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state": models.CanvasNodeExecutionStateStarted,
	}).Error)

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "noop",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	require.NoError(t, database.Conn().Delete(&canvasNodes[0]).Error)

	err := worker.LockAndProcessRequest(request)
	require.NoError(t, err)

	assert.True(t, cancelCalled, "component Cancel should be invoked")

	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	var updatedExecution models.CanvasNodeExecution
	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)
}

func Test__NodeRequestWorker_DoesNotProcessDeletedWorkflowRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a schedule trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for the trigger.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Soft delete the entire workflow.
	//
	require.NoError(t, database.Conn().Delete(&canvas).Error)

	//
	// Verify that ListNodeRequests does not return the request for the deleted workflow.
	//
	requests, err := models.ListNodeRequests()
	require.NoError(t, err)

	// Check that our request is not in the list
	found := false
	for _, req := range requests {
		if req.ID == request.ID {
			found = true
			break
		}
	}
	assert.False(t, found, "Request for deleted workflow should not be returned by ListNodeRequests")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_DoesNotProcessSoftDeletedOrganizationRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))

	requests, err := models.ListNodeRequests()
	require.NoError(t, err)
	for _, pending := range requests {
		assert.NotEqual(t, request.ID, pending.ID)
	}

	require.NoError(t, worker.LockAndProcessRequest(request))

	var updatedRequest models.CanvasNodeRequest
	require.NoError(t, database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error)
	assert.Equal(t, models.NodeExecutionRequestStatePending, updatedRequest.State)

	eventCount, err := models.CountCanvasEvents(canvas.ID, triggerNode)
	require.NoError(t, err)
	assert.Zero(t, eventCount)
	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_SkipsHookForFinishedExecution(t *testing.T) {
	//
	// A poll-style hook can be requested after the execution is already finished
	// (for example, a poll scheduled before an incoming webhook finished it).
	// A finished execution is terminal, so the worker must not invoke the hook
	// again - re-saving the execution would move its updated_at, which is the
	// timestamp surfaced as the execution's finished_at (issue #6126). The
	// request is completed regardless.
	//
	hookInvoked := false
	componentName := "finished_poll_" + uuid.New().String()
	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  componentName,
		Hooks: []core.Hook{{Name: "poll", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			hookInvoked = true
			return nil
		},
	}))

	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})

	finishedAt := time.Now().Add(-10 * time.Minute)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": finishedAt,
	}).Error)

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "poll",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	err := worker.LockAndProcessRequest(request)
	require.NoError(t, err)

	assert.False(t, hookInvoked, "hook must not be invoked for an already finished execution")

	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State,
		"request must be completed even when the hook is skipped")

	var updatedExecution models.CanvasNodeExecution
	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	require.NotNil(t, updatedExecution.UpdatedAt)
	assert.WithinDuration(t, finishedAt, *updatedExecution.UpdatedAt, time.Second,
		"finished execution updated_at must not be overwritten by a skipped hook")
}

func Test__NodeRequestWorker_UpdatesUpdatedAtForRunningExecution(t *testing.T) {
	//
	// Counterpart to the guard above: updated_at is only preserved for executions
	// that were ALREADY finished before the hook ran. A hook on a still-running
	// execution must let updated_at advance as usual, so the timestamp keeps
	// reflecting the last activity. This pins that boundary so the fix cannot be
	// widened into omitting updated_at for every hook invocation.
	//
	componentName := "running_poll_" + uuid.New().String()
	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  componentName,
		Hooks: []core.Hook{{Name: "poll", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			return nil
		},
	}))

	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, r.GitProvider, "", r.AuthService)

	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})

	staleUpdatedAt := time.Now().Add(-10 * time.Minute)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateStarted,
		"updated_at": staleUpdatedAt,
	}).Error)

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "poll",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	err := worker.LockAndProcessRequest(request)
	require.NoError(t, err)

	var updatedExecution models.CanvasNodeExecution
	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	require.NotNil(t, updatedExecution.UpdatedAt)
	assert.True(t, updatedExecution.UpdatedAt.After(staleUpdatedAt),
		"running execution updated_at must advance when a hook runs")
}
