package workers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__NodeRequestWorker_InvokeTriggerAction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
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

func Test__NodeRequestWorker_InvokeNodeComponentActionWithoutExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
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
	assert.Contains(t, err.Error(), "action 'non-existent-action' not found for component 'noop'")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_PreventsConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
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
		worker1 := NewNodeRequestWorker(r.Encryptor, r.Registry)
		results <- worker1.LockAndProcessRequest(request)
	}()

	go func() {
		worker2 := NewNodeRequestWorker(r.Encryptor, r.Registry)
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
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
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
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
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
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
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
	assert.Contains(t, err.Error(), "trigger not found")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_NonExistentAction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
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
	assert.Contains(t, err.Error(), "action 'non-existent-action' not found")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_DoesNotProcessDeletedNodeRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a schedule trigger node.
	//
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
	// Soft delete the workflow node.
	//
	require.NoError(t, database.Conn().Delete(&canvasNodes[0]).Error)

	//
	// Verify that ListNodeRequests does not return the request for the deleted node.
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
	assert.False(t, found, "Request for deleted node should not be returned by ListNodeRequests")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_DoesNotProcessDeletedWorkflowRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
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
