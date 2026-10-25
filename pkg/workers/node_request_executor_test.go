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

func Test__NodeRequestExecutor_RunNow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "schedule", Type: models.NodeTypeTrigger,
		Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
		Configuration: datatypes.NewJSONType(map[string]any{
			"type": "days", "daysInterval": 1, "hour": 12, "minute": 0,
		}),
	}}, nil)

	for _, name := range []string{"future request", "locked request", "completed request", "failed hook"} {
		t.Run(name, func(t *testing.T) {
			request := models.CanvasNodeRequest{
				ID: uuid.New(), WorkflowID: canvas.ID, NodeID: "schedule",
				Type: models.NodeRequestTypeInvokeAction, State: models.NodeExecutionRequestStatePending,
				RunAt: time.Now().Add(time.Hour),
				Spec:  datatypes.NewJSONType(models.NodeExecutionRequestSpec{InvokeAction: &models.InvokeAction{ActionName: "emitEvent"}}),
			}
			if name == "completed request" {
				request.State = models.NodeExecutionRequestStateCompleted
			}
			if name == "failed hook" {
				request.Spec = datatypes.NewJSONType(models.NodeExecutionRequestSpec{InvokeAction: &models.InvokeAction{ActionName: "missing"}})
			}
			require.NoError(t, db.Create(&request).Error)
			if name == "locked request" {
				tx := db.Begin()
				require.NoError(t, tx.Error)
				defer tx.Rollback()
				_, err := models.LockNodeRequestNow(tx, request.ID)
				require.NoError(t, err)
			}
			var before int64
			require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&before).Error)
			done := make(chan error, 1)
			go func() { done <- executor.RunNow(request) }()
			select {
			case err := <-done:
				if name == "failed hook" {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("RunNow must skip requests held by another executor")
			}
			updated, err := models.FindCanvasNodeRequest(db, request.ID)
			require.NoError(t, err)
			var after int64
			require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&after).Error)
			if name == "future request" {
				assert.Equal(t, models.NodeExecutionRequestStateCompleted, updated.State)
				assert.Equal(t, before+1, after)
				return
			}
			assert.Equal(t, request.State, updated.State)
			assert.WithinDuration(t, request.RunAt, updated.RunAt, time.Microsecond)
			assert.Equal(t, before, after)
		})
	}
}

func Test__NodeRequestExecutor_InvokeTriggerAction_DefersRunTitleResolutionUntilEventEmit(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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

	err := executor.Execute(request)
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

func Test__NodeRequestExecutor_InvokeNodeComponentActionWithoutExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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

	err := executor.Execute(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook non-existent-action not found for action noop")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_PreventsConcurrentProcessing(t *testing.T) {
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
	// Have two workers call Execute concurrently on the same request.
	// Execute uses a transaction with locking, so only one should actually process.
	//
	results := make(chan error, 2)

	//
	// Create two workers and have them try to process the request concurrently.
	//
	go func() {
		executor1 := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
		results <- executor1.Execute(request)
	}()

	go func() {
		executor2 := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
		results <- executor2.Execute(request)
	}()

	// Collect results - both should succeed (return nil)
	// because Execute returns nil when it can't acquire the lock
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
	// Verify that exactly one workflow event was emitted (proving only one executor processed it).
	//
	eventCount, err := models.CountCanvasEvents(database.Conn(), canvas.ID, triggerNode)
	require.NoError(t, err)
	assert.Equal(t, int64(1), eventCount, "Expected exactly 1 workflow event, but found %d", eventCount)

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_UnsupportedRequestType(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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
	err := executor.Execute(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported node execution request type")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_MissingInvokeActionSpec(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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
	err := executor.Execute(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec is not specified")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_NonExistentTrigger(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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
	err := executor.Execute(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trigger non-existent-trigger not registered")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_NonExistentAction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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
	err := executor.Execute(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook non-existent-action not found for trigger schedule")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_CompletesDeletedNodeRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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

	err = executor.Execute(request)
	require.NoError(t, err)

	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	eventCount, err := models.CountCanvasEvents(database.Conn(), canvas.ID, triggerNode)
	require.NoError(t, err)
	assert.Zero(t, eventCount)

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_CancelsExecutionForDeletedNodeRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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

	err := executor.Execute(request)
	require.NoError(t, err)

	var updatedExecution models.CanvasNodeExecution
	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)

	terminator := NewExecutionTerminator("", r.AuthService, r.Encryptor, r.Registry)
	require.NoError(t, terminator.LockAndCancelExecution(updatedExecution))

	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)

	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
}

func Test__NodeRequestExecutor_CompletesRequestForFinishedExecutionWithoutInvokingHook(t *testing.T) {
	hookCalled := false
	componentName := "finished_execution_request_" + uuid.New().String()
	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  componentName,
		Hooks: []core.Hook{{Name: "poll", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			hookCalled = true
			return nil
		},
	}))

	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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

	err := executor.Execute(request)
	require.NoError(t, err)

	assert.False(t, hookCalled)

	var updatedRequest models.CanvasNodeRequest
	require.NoError(t, database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	require.NotNil(t, updatedExecution.UpdatedAt)
	assert.WithinDuration(t, finishedAt, *updatedExecution.UpdatedAt, time.Second)
}

func Test__NodeRequestExecutor_CompletesRequestForCancellingExecutionWithoutInvokingHook(t *testing.T) {
	hookCalled := false
	componentName := "cancelling_execution_request_" + uuid.New().String()
	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  componentName,
		Hooks: []core.Hook{{Name: "poll", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			hookCalled = true
			return nil
		},
	}))

	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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

	cancellingAt := time.Now().Add(-10 * time.Minute)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateCancelling,
		"updated_at": cancellingAt,
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

	err := executor.Execute(request)
	require.NoError(t, err)

	assert.False(t, hookCalled)

	var updatedRequest models.CanvasNodeRequest
	require.NoError(t, database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)
	require.NotNil(t, updatedExecution.UpdatedAt)
	assert.WithinDuration(t, cancellingAt, *updatedExecution.UpdatedAt, time.Second)
}

func Test__NodeRequestExecutor_DoesNotSaveStaleExecutionAfterHookFindsPersistedExecutionFinished(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	componentName := "stale_execution_after_hook_" + uuid.New().String()
	var executionID uuid.UUID
	finishedAt := time.Now().Add(-10 * time.Minute)
	r.Registry.Actions[componentName] = impl.NewDummyAction(impl.DummyActionOptions{
		Name:  componentName,
		Hooks: []core.Hook{{Name: "poll", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			err := database.Conn().Model(&models.CanvasNodeExecution{}).
				Where("id = ?", executionID).
				Updates(map[string]any{
					"state":      models.CanvasNodeExecutionStateFinished,
					"result":     models.CanvasNodeExecutionResultPassed,
					"updated_at": finishedAt,
				}).Error
			require.NoError(t, err)

			return ctx.ExecutionState.Pass()
		},
	})

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
	require.NoError(t, database.Conn().Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
	executionID = execution.ID

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

	err := executor.Execute(request)
	require.NoError(t, err)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, updatedExecution.Result)
	require.NotNil(t, updatedExecution.UpdatedAt)
	assert.WithinDuration(t, finishedAt, *updatedExecution.UpdatedAt, time.Second)

	var updatedRequest models.CanvasNodeRequest
	require.NoError(t, database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
}

func Test__NodeRequestExecutor_CancelsExecutionForDeletedNodeWhenComponentCancelFails(t *testing.T) {
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
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

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

	err := executor.Execute(request)
	require.NoError(t, err)

	assert.False(t, cancelCalled, "NodeRequestExecutor should not invoke component Cancel directly")

	var updatedExecution models.CanvasNodeExecution
	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)

	terminator := NewExecutionTerminator("", r.AuthService, r.Encryptor, r.Registry)
	require.NoError(t, terminator.LockAndCancelExecution(updatedExecution))

	assert.True(t, cancelCalled, "component Cancel should be invoked by ExecutionTerminator")

	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)

	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
}

func Test__NodeRequestExecutor_DoesNotProcessDeletedWorkflowRequests(t *testing.T) {
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

func Test__NodeRequestExecutor_DoesNotProcessSoftDeletedOrganizationRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
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

	require.NoError(t, executor.Execute(request))

	var updatedRequest models.CanvasNodeRequest
	require.NoError(t, database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error)
	assert.Equal(t, models.NodeExecutionRequestStatePending, updatedRequest.State)

	eventCount, err := models.CountCanvasEvents(database.Conn(), canvas.ID, triggerNode)
	require.NoError(t, err)
	assert.Zero(t, eventCount)
	assert.False(t, executionConsumer.HasReceivedMessage())
}
