package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__WorkflowNodeQueueWorker_ComponentNodeQueueIsProcessed(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowNodeQueueWorker(r.Registry)

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
	err = worker.LockAndProcessNode(*node)
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
}

func Test__WorkflowNodeQueueWorker_BlueprintNodeQueueIsProcessed(t *testing.T) {
	r := support.Setup(t)

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
	err = worker.LockAndProcessNode(*node)
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
}

func Test__WorkflowNodeQueueWorker_PicksOldestQueueItem(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowNodeQueueWorker(r.Registry)

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

	err = worker.LockAndProcessNode(*node)
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
}

func Test__WorkflowNodeQueueWorker_EmptyQueue(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowNodeQueueWorker(r.Registry)

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
	err = worker.LockAndProcessNode(*node)
	require.NoError(t, err)

	// Verify no executions were created
	executions, err := models.ListNodeExecutions(workflow.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	assert.Len(t, executions, 0)

	// Verify node state is still ready
	node, err = models.FindWorkflowNode(database.Conn(), workflow.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateReady, node.State)
}

func Test__WorkflowNodeQueueWorker_PreventsConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

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
		results <- worker1.LockAndProcessNode(*node)
	}()

	go func() {
		worker2 := NewWorkflowNodeQueueWorker(r.Registry)
		results <- worker2.LockAndProcessNode(*node)
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
}
