package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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

func Test__WorkflowNodeQueueWorker_ComponentNodeQueueIsProcessed(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowNodeQueueWorker(r.Registry)
	logger := log.NewEntry(log.New())

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.WorkflowQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	//
	// Create a simple workflow with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	//
	// Create a root event and a queue item for the component node.
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	support.CreateWorkflowQueueItem(t, workflow.ID, componentNode, rootEvent.ID, rootEvent.ID)

	//
	// Verify initial state - node should be ready and queue should have 1 item.
	//
	node, err := models.FindWorkflowNode(database.Conn(), workflow.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateReady, node.State)

	queueItems, err := models.ListNodeQueueItems(workflow.ID, componentNode, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)

	//
	// Process the node and verify the happy path:
	// - Pending execution is created
	// - Node state is updated to processing
	// - Queue item is deleted
	//
	err = worker.LockAndProcessNode(logger, *node)
	require.NoError(t, err)

	// Verify execution was created with pending state
	executions, err := models.ListNodeExecutions(workflow.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, models.WorkflowNodeExecutionStatePending, executions[0].State)
	assert.Equal(t, rootEvent.ID, executions[0].EventID)
	assert.Equal(t, rootEvent.ID, executions[0].RootEventID)

	// Verify node state was updated to processing
	node, err = models.FindWorkflowNode(database.Conn(), workflow.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateProcessing, node.State)

	// Verify queue item was deleted
	queueItems, err = models.ListNodeQueueItems(workflow.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, queueItems, 0)

	assert.True(t, executionConsumer.HasReceivedMessage())
	assert.True(t, queueConsumedConsumer.HasReceivedMessage())
}

func Test__WorkflowNodeQueueWorker_BlueprintNodeQueueIsProcessed(t *testing.T) {
	r := support.Setup(t)
	logger := log.NewEntry(log.New())

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.WorkflowQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	//
	// Create a simple blueprint with two sequential nodes
	//
	blueprint := support.CreateBlueprint(
		t,
		r.Organization.ID,
		[]models.Node{
			{
				ID:   "noop1",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
			},
			{
				ID:   "noop2",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
			},
		},
		[]models.Edge{
			{SourceID: "noop1", TargetID: "noop2", Channel: "default"},
		},
		[]models.BlueprintOutputChannel{
			{
				Name:              "default",
				NodeID:            "noop2",
				NodeOutputChannel: "default",
			},
		},
	)

	//
	// Create a simple workflow with a trigger and a blueprint node.
	//
	triggerNode := "trigger-1"
	blueprintNode := "blueprint-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: blueprintNode,
				Type:   models.NodeTypeBlueprint,
				Ref:    datatypes.NewJSONType(models.NodeRef{Blueprint: &models.BlueprintRef{ID: blueprint.ID.String()}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: blueprintNode, Channel: "default"},
		},
	)

	//
	// Create a root event and a queue item for the blueprint node.
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	support.CreateWorkflowQueueItem(t, workflow.ID, blueprintNode, rootEvent.ID, rootEvent.ID)

	//
	// Verify initial state - node should be ready and queue should have 1 item.
	//
	node, err := models.FindWorkflowNode(database.Conn(), workflow.ID, blueprintNode)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateReady, node.State)

	queueItems, err := models.ListNodeQueueItems(workflow.ID, blueprintNode, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)

	//
	// Process the node and verify the happy path:
	// - Pending execution is created
	// - Node state is updated to processing
	// - Queue item is deleted
	//
	worker := NewWorkflowNodeQueueWorker(r.Registry)
	err = worker.LockAndProcessNode(logger, *node)
	require.NoError(t, err)

	// Verify execution was created with pending state
	executions, err := models.ListNodeExecutions(workflow.ID, blueprintNode, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, models.WorkflowNodeExecutionStatePending, executions[0].State)
	assert.Equal(t, rootEvent.ID, executions[0].EventID)
	assert.Equal(t, rootEvent.ID, executions[0].RootEventID)

	// Verify node state was updated to processing
	node, err = models.FindWorkflowNode(database.Conn(), workflow.ID, blueprintNode)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateProcessing, node.State)

	// Verify queue item was deleted
	queueItems, err = models.ListNodeQueueItems(workflow.ID, blueprintNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, queueItems, 0)

	assert.True(t, executionConsumer.HasReceivedMessage())
	assert.True(t, queueConsumedConsumer.HasReceivedMessage())
}

func Test__WorkflowNodeQueueWorker_PicksOldestQueueItem(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowNodeQueueWorker(r.Registry)
	logger := log.NewEntry(log.New())

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.WorkflowQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	//
	// Create a simple workflow with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{NodeID: triggerNode, Type: models.NodeTypeTrigger},
			{NodeID: componentNode, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	//
	// Create three queue items with different timestamps.
	// We'll manually insert them with specific created_at times.
	//
	oldTime := time.Now().Add(-10 * time.Minute)
	midTime := time.Now().Add(-5 * time.Minute)
	newTime := time.Now()

	oldEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	midEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	newEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)

	// Create queue items with specific timestamps
	oldQueueItem := models.WorkflowNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  workflow.ID,
		NodeID:      componentNode,
		RootEventID: oldEvent.ID,
		EventID:     oldEvent.ID,
		CreatedAt:   &oldTime,
	}
	midQueueItem := models.WorkflowNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  workflow.ID,
		NodeID:      componentNode,
		RootEventID: midEvent.ID,
		EventID:     midEvent.ID,
		CreatedAt:   &midTime,
	}
	newQueueItem := models.WorkflowNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  workflow.ID,
		NodeID:      componentNode,
		RootEventID: newEvent.ID,
		EventID:     newEvent.ID,
		CreatedAt:   &newTime,
	}

	require.NoError(t, database.Conn().Create(&oldQueueItem).Error)
	require.NoError(t, database.Conn().Create(&midQueueItem).Error)
	require.NoError(t, database.Conn().Create(&newQueueItem).Error)

	//
	// Process the node and verify that the oldest queue item was picked.
	//
	node, err := models.FindWorkflowNode(database.Conn(), workflow.ID, componentNode)
	require.NoError(t, err)

	err = worker.LockAndProcessNode(logger, *node)
	require.NoError(t, err)

	// Verify the execution was created with the oldest event
	executions, err := models.ListNodeExecutions(workflow.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, oldEvent.ID, executions[0].EventID)

	// Verify the oldest queue item was deleted, but the other two remain
	queueItems, err := models.ListNodeQueueItems(workflow.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, queueItems, 2)

	// Verify the remaining items are the mid and new ones
	eventIDs := []uuid.UUID{queueItems[0].EventID, queueItems[1].EventID}
	assert.Contains(t, eventIDs, midEvent.ID)
	assert.Contains(t, eventIDs, newEvent.ID)
	assert.NotContains(t, eventIDs, oldEvent.ID)

	assert.True(t, executionConsumer.HasReceivedMessage())
	assert.True(t, queueConsumedConsumer.HasReceivedMessage())
}

func Test__WorkflowNodeQueueWorker_EmptyQueue(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowNodeQueueWorker(r.Registry)
	logger := log.NewEntry(log.New())

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.WorkflowQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	//
	// Create a simple workflow with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "noop"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{NodeID: triggerNode, Type: models.NodeTypeTrigger},
			{NodeID: componentNode, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	//
	// Don't create any queue items - the queue is empty.
	//
	node, err := models.FindWorkflowNode(database.Conn(), workflow.ID, componentNode)
	require.NoError(t, err)

	//
	// Process the node with an empty queue - this should succeed but do nothing.
	//
	err = worker.LockAndProcessNode(logger, *node)
	require.NoError(t, err)

	// Verify no executions were created
	executions, err := models.ListNodeExecutions(workflow.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	assert.Len(t, executions, 0)

	// Verify node state is still ready
	node, err = models.FindWorkflowNode(database.Conn(), workflow.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateReady, node.State)

	assert.False(t, executionConsumer.HasReceivedMessage())
	assert.False(t, queueConsumedConsumer.HasReceivedMessage())
}

func Test__WorkflowNodeQueueWorker_PreventsConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.WorkflowQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	//
	// Create a simple workflow with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "noop"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{NodeID: triggerNode, Type: models.NodeTypeTrigger},
			{NodeID: componentNode, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	//
	// Create a root event and a queue item for the component node.
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	support.CreateWorkflowQueueItem(t, workflow.ID, componentNode, rootEvent.ID, rootEvent.ID)

	node, err := models.FindWorkflowNode(database.Conn(), workflow.ID, componentNode)
	require.NoError(t, err)

	//
	// Have two workers call LockAndProcessNode concurrently on the same node.
	// LockAndProcessNode uses a transaction with SKIP LOCKED, so only one should actually process.
	//
	results := make(chan error, 2)

	//
	// Create two workers and have them try to process the node concurrently.
	//
	go func() {
		worker1 := NewWorkflowNodeQueueWorker(r.Registry)
		logger := log.NewEntry(log.New())
		results <- worker1.LockAndProcessNode(logger, *node)
	}()

	go func() {
		worker2 := NewWorkflowNodeQueueWorker(r.Registry)
		logger := log.NewEntry(log.New())
		results <- worker2.LockAndProcessNode(logger, *node)
	}()

	// Collect results - both should succeed (return nil)
	// because LockAndProcessNode returns nil when it can't acquire the lock
	result1 := <-results
	result2 := <-results
	assert.NoError(t, result1)
	assert.NoError(t, result2)

	//
	// Verify only one execution was created (not two).
	// This proves that only one worker actually processed the node.
	//
	executions, err := models.ListNodeExecutions(workflow.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	assert.Len(t, executions, 1, "Only one execution should be created, not two")

	//
	// Verify the queue item was deleted.
	//
	queueItems, err := models.ListNodeQueueItems(workflow.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, queueItems, 0, "Queue item should be deleted")

	assert.True(t, executionConsumer.HasReceivedMessage())
	assert.True(t, queueConsumedConsumer.HasReceivedMessage())
}

func Test__WorkflowNodeQueueWorker_ConfigurationBuildFailure(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowNodeQueueWorker(r.Registry)
	logger := log.NewEntry(log.New())

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple workflow with a trigger and a component node.
	// The component node will have a configuration with an invalid expression.
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	workflow, nodes := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
				// Invalid expression that will fail during Build()
				Configuration: datatypes.NewJSONType(map[string]any{
					"field": "{{ $[\"nonexistent-node\"].data }}",
				}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	//
	// Create a root event and a queue item for the component node.
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	support.CreateWorkflowQueueItem(t, workflow.ID, componentNode, rootEvent.ID, rootEvent.ID)

	//
	// Verify initial state - node should be ready and queue should have 1 item.
	//
	node, err := models.FindWorkflowNode(database.Conn(), workflow.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateReady, node.State)

	queueItems, err := models.ListNodeQueueItems(workflow.ID, componentNode, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)

	//
	// Process the node - this should handle the configuration build failure:
	// - Failed execution should be created
	// - Node state should remain ready (not updated to processing)
	// - Queue item should be deleted
	//
	err = worker.LockAndProcessNode(logger, *node)
	require.NoError(t, err)

	// Verify execution was created with finished state and failed result
	executions, err := models.ListNodeExecutions(workflow.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, executions[0].State)
	assert.Equal(t, models.WorkflowNodeExecutionResultFailed, executions[0].Result)
	assert.Equal(t, models.WorkflowNodeExecutionResultReasonError, executions[0].ResultReason)
	assert.Contains(t, executions[0].ResultMessage, "field")
	assert.Equal(t, rootEvent.ID, executions[0].EventID)
	assert.Equal(t, rootEvent.ID, executions[0].RootEventID)
	assert.Equal(t, nodes[1].Configuration, executions[0].Configuration)

	// Verify node state is still ready (not updated to processing)
	node, err = models.FindWorkflowNode(database.Conn(), workflow.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateReady, node.State)

	// Verify queue item was deleted
	queueItems, err = models.ListNodeQueueItems(workflow.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, queueItems, 0)

	// Verify execution message was published
	assert.True(t, executionConsumer.HasReceivedMessage())
}

func Test__WorkflowNodeQueueWorker_MergeComponentReturnsNilExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowNodeQueueWorker(r.Registry)
	logger := log.NewEntry(log.New())

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.WorkflowQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	//
	// Create a workflow with two source nodes feeding into a merge node.
	// This simulates a scenario where merge needs to wait for multiple inputs.
	//
	source1Node := "source-1"
	source2Node := "source-2"
	mergeNode := "merge-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: source1Node,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: source2Node,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: mergeNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}}),
			},
		},
		[]models.Edge{
			{SourceID: source1Node, TargetID: mergeNode, Channel: "default"},
			{SourceID: source2Node, TargetID: mergeNode, Channel: "default"},
		},
	)

	//
	// Create a root event and queue item for the merge node from only one source.
	// Since merge expects 2 inputs but only gets 1, it should return nil execution.
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, source1Node, "default", nil)
	support.CreateWorkflowQueueItem(t, workflow.ID, mergeNode, rootEvent.ID, rootEvent.ID)

	//
	// Verify initial state - node should be ready and queue should have 1 item.
	//
	node, err := models.FindWorkflowNode(database.Conn(), workflow.ID, mergeNode)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateReady, node.State)

	queueItems, err := models.ListNodeQueueItems(workflow.ID, mergeNode, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)

	//
	// Process the node - merge should return nil execution because it's waiting for more inputs.
	//
	err = worker.LockAndProcessNode(logger, *node)
	require.NoError(t, err)

	node, err = models.FindWorkflowNode(database.Conn(), workflow.ID, mergeNode)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateReady, node.State)

	queueItems, err = models.ListNodeQueueItems(workflow.ID, mergeNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, queueItems, 0, "Queue item should be processed even when returning nil execution")

	assert.False(t, executionConsumer.HasReceivedMessage())
	assert.True(t, queueConsumedConsumer.HasReceivedMessage())
}

func Test__WorkflowNodeQueueWorker_ConfigurationBuildFailure_PropagateToParent(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowNodeQueueWorker(r.Registry)
	logger := log.NewEntry(log.New())

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.WorkflowExecutionRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a blueprint with two nodes - the second one has an invalid configuration.
	// This way, the first node will process successfully and create the parent execution,
	// and the second node will fail during configuration build.
	//
	blueprint := support.CreateBlueprint(
		t,
		r.Organization.ID,
		[]models.Node{
			{
				ID:   "noop1",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
			},
			{
				ID:   "noop2",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
				// Invalid expression that will fail during Build()
				Configuration: map[string]any{
					"field": "{{ $[\"nonexistent-node\"].data }}",
				},
			},
		},
		[]models.Edge{
			{SourceID: "noop1", TargetID: "noop2", Channel: "default"},
		},
		[]models.BlueprintOutputChannel{
			{
				Name:              "default",
				NodeID:            "noop2",
				NodeOutputChannel: "default",
			},
		},
	)

	//
	// Create a workflow with the blueprint node.
	//
	triggerNode := "trigger-1"
	blueprintNode := "blueprint-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: blueprintNode,
				Type:   models.NodeTypeBlueprint,
				Ref:    datatypes.NewJSONType(models.NodeRef{Blueprint: &models.BlueprintRef{ID: blueprint.ID.String()}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: blueprintNode, Channel: "default"},
		},
	)

	//
	// Create a root event and process the blueprint node to create a parent execution.
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	support.CreateWorkflowQueueItem(t, workflow.ID, blueprintNode, rootEvent.ID, rootEvent.ID)

	node, err := models.FindWorkflowNode(database.Conn(), workflow.ID, blueprintNode)
	require.NoError(t, err)

	err = worker.LockAndProcessNode(logger, *node)
	require.NoError(t, err)

	// Get the parent execution that was created
	parentExecutions, err := models.ListNodeExecutions(workflow.ID, blueprintNode, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, parentExecutions, 1)
	parentExecution := parentExecutions[0]
	assert.Equal(t, models.WorkflowNodeExecutionStatePending, parentExecution.State)

	//
	// Now we need to simulate the first child node (noop1) executing successfully.
	// Create and pass the first child execution, which will emit an event.
	//
	firstChildExecution, err := models.CreatePendingChildExecution(
		database.Conn(),
		&parentExecution,
		"noop1",
		map[string]any{},
	)
	require.NoError(t, err)

	// Pass the first child execution to emit an event
	events, err := firstChildExecution.Pass(map[string][]any{
		"default": {map[string]any{"data": "test"}},
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	firstChildEvent := events[0]

	//
	// Now create a queue item for the second child node (noop2) using the event
	// emitted by noop1. This node has the invalid configuration and should fail
	// during configuration build.
	//
	secondChildNodeID := blueprintNode + ":noop2"
	support.CreateWorkflowQueueItem(t, workflow.ID, secondChildNodeID, rootEvent.ID, firstChildEvent.ID)

	childNode, err := models.FindWorkflowNode(database.Conn(), workflow.ID, secondChildNodeID)
	require.NoError(t, err)

	//
	// Process the child node - this should:
	// 1. Create a failed child execution with ParentExecutionID set
	// 2. Propagate the failure to the parent execution
	//
	err = worker.LockAndProcessNode(logger, *childNode)
	require.NoError(t, err)

	//
	// Verify the child execution was created with:
	// - Finished state and failed result
	// - ParentExecutionID pointing to the parent execution
	//
	childExecutions, err := models.ListNodeExecutions(workflow.ID, secondChildNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, childExecutions, 1)
	childExecution := childExecutions[0]

	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, childExecution.State)
	assert.Equal(t, models.WorkflowNodeExecutionResultFailed, childExecution.Result)
	assert.Equal(t, models.WorkflowNodeExecutionResultReasonError, childExecution.ResultReason)
	assert.Contains(t, childExecution.ResultMessage, "field")
	require.NotNil(t, childExecution.ParentExecutionID, "Child execution should have ParentExecutionID set")
	assert.Equal(t, parentExecution.ID, *childExecution.ParentExecutionID)

	//
	// Verify the parent execution was also failed (propagated from child).
	//
	updatedParent, err := models.FindNodeExecution(workflow.ID, parentExecution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, updatedParent.State)
	assert.Equal(t, models.WorkflowNodeExecutionResultFailed, updatedParent.Result)
	assert.Equal(t, models.WorkflowNodeExecutionResultReasonError, updatedParent.ResultReason)
}
