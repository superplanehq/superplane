package workers

import (
	"testing"
	"time"

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
	updatedRequest, err := models.FindNodeRequest(request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
	assert.Equal(t, models.NodeExecutionRequestResultPassed, updatedRequest.Result)
	assert.Equal(t, 1, updatedRequest.Attempts)
	assert.Empty(t, updatedRequest.ResultMessage)

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
	updatedRequest, err := models.FindNodeRequest(request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
	assert.Equal(t, models.NodeExecutionRequestResultPassed, updatedRequest.Result)
	assert.Equal(t, 1, updatedRequest.Attempts)
	assert.Empty(t, updatedRequest.ResultMessage)

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
	require.NoError(t, err)

	//
	// Reload request and verify its state.
	//
	updatedRequest, err := models.FindNodeRequest(request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
	assert.Equal(t, models.NodeExecutionRequestResultFailed, updatedRequest.Result)
	assert.Equal(t, 1, updatedRequest.Attempts)
	assert.Contains(t, updatedRequest.ResultMessage, "spec is not specified")

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
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := worker.LockAndProcessRequest(request)
	require.NoError(t, err)

	updatedRequest, err := models.FindNodeRequest(request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
	assert.Equal(t, models.NodeExecutionRequestResultFailed, updatedRequest.Result)
	assert.Equal(t, 1, updatedRequest.Attempts)
	assert.Contains(t, updatedRequest.ResultMessage, "trigger non-existent-trigger not registered")

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
	require.NoError(t, err)

	//
	// Reload request and verify its state.
	//
	updatedRequest, err := models.FindNodeRequest(request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
	assert.Equal(t, models.NodeExecutionRequestResultFailed, updatedRequest.Result)
	assert.Equal(t, 1, updatedRequest.Attempts)
	assert.Contains(t, updatedRequest.ResultMessage, "action 'non-existent-action' not found")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestWorker_RetryStrategy_AttemptsExhausted(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Encryptor, r.Registry)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a schedule trigger node with an invalid configuration to force action failure.
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
					"type": "days",
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
		RetryStrategy: datatypes.NewJSONType(models.RetryStrategy{
			Type: models.RetryStrategyTypeConstant,
			Constant: &models.ConstantRetryStrategy{
				MaxAttempts: 2,
				Delay:       1,
			},
		}),
		Attempts: 0,
		RunAt:    time.Now(),
		State:    models.NodeExecutionRequestStatePending,
	}

	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// First try, request should remain in pending state.
	//
	err := worker.LockAndProcessRequest(request)
	require.NoError(t, err)
	updatedRequest, err := models.FindNodeRequest(request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStatePending, updatedRequest.State)
	assert.Equal(t, 1, updatedRequest.Attempts)
	assert.Empty(t, updatedRequest.Result)
	assert.Empty(t, updatedRequest.ResultMessage)

	//
	// Second try, request should be completed and failed.
	//
	time.Sleep(20 * time.Millisecond)
	err = worker.LockAndProcessRequest(*updatedRequest)
	require.NoError(t, err)
	updatedRequest, err = models.FindNodeRequest(request.ID)
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
	assert.Equal(t, models.NodeExecutionRequestResultFailed, updatedRequest.Result)
	assert.Equal(t, 2, updatedRequest.Attempts)
	assert.Contains(t, updatedRequest.ResultMessage, "max attempts reached")

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
