package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"gorm.io/datatypes"
)

func Test__NodeQueueWorker_ComponentNodeQueueIsProcessed(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)
	logger := log.NewEntry(log.New())

	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	//
	// Create a simple canvas with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	support.CreateQueueItem(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)

	//
	// Verify initial state - node should be ready and queue should have 1 item.
	//
	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateReady, node.State)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)

	//
	// Process the node and verify the happy path:
	// - Pending execution is created
	// - Node state is updated to processing
	// - Queue item is deleted
	//
	err = worker.LockAndProcessNode(logger, *node, time.Now())
	require.NoError(t, err)

	// Verify execution was created with pending state
	executions, err := models.ListNodeExecutions(canvas.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, models.CanvasNodeExecutionStatePending, executions[0].State)
	assert.Equal(t, rootEvent.ID, executions[0].EventID)
	assert.Equal(t, rootEvent.ID, executions[0].RootEventID)

	// Verify node state was updated to processing
	node, err = models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateProcessing, node.State)

	// Verify queue item was deleted
	queueItems, err = models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, queueItems, 0)

	assert.True(t, executionConsumer.HasReceivedMessage())
	assert.True(t, queueConsumedConsumer.HasReceivedMessage())
}

func Test__NodeQueueWorker_DoesNotProcessQueueForSoftDeletedOrganization(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)
	logger := log.NewEntry(log.New())

	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNode, Type: models.NodeTypeTrigger},
			{NodeID: componentNode, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	support.CreateQueueItem(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)

	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))

	nodes, err := models.ListCanvasNodesReady()
	require.NoError(t, err)
	for _, node := range nodes {
		assert.False(t, node.WorkflowID == canvas.ID && node.NodeID == componentNode)
	}

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)

	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	executions, err := models.ListNodeExecutions(canvas.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, executions)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, queueItems, 1)

	assert.False(t, executionConsumer.HasReceivedMessage())
	assert.False(t, queueConsumedConsumer.HasReceivedMessage())
}

func Test__NodeQueueWorker_SkipsMissingNode(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)

	err := worker.tryProcessReadyNode(uuid.New(), "deleted-node", time.Now())
	require.NoError(t, err)
}

func Test__NodeQueueWorker_PicksOldestQueueItem(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)
	logger := log.NewEntry(log.New())

	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	//
	// Create a simple canvas with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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

	oldEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	midEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	newEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)

	// Create queue items with specific timestamps
	oldQueueItem := models.CanvasNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		RootEventID: oldEvent.ID,
		EventID:     oldEvent.ID,
		CreatedAt:   &oldTime,
	}
	midQueueItem := models.CanvasNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		RootEventID: midEvent.ID,
		EventID:     midEvent.ID,
		CreatedAt:   &midTime,
	}
	newQueueItem := models.CanvasNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
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
	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)

	err = worker.LockAndProcessNode(logger, *node, time.Now())
	require.NoError(t, err)

	// Verify the execution was created with the oldest event
	executions, err := models.ListNodeExecutions(canvas.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, oldEvent.ID, executions[0].EventID)

	// Verify the oldest queue item was deleted, but the other two remain
	queueItems, err := models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
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

func Test__NodeQueueWorker_EmptyQueue(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)
	logger := log.NewEntry(log.New())

	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	//
	// Create a simple canvas with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "noop"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)

	//
	// Process the node with an empty queue - this should succeed but do nothing.
	//
	err = worker.LockAndProcessNode(logger, *node, time.Now())
	require.NoError(t, err)

	// Verify no executions were created
	executions, err := models.ListNodeExecutions(canvas.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	assert.Len(t, executions, 0)

	// Verify node state is still ready
	node, err = models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateReady, node.State)

	assert.False(t, executionConsumer.HasReceivedMessage())
	assert.False(t, queueConsumedConsumer.HasReceivedMessage())
}

func Test__NodeQueueWorker_PreventsConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	queueConsumedConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemConsumedRoutingKey)
	executionConsumer.Start()
	queueConsumedConsumer.Start()
	defer executionConsumer.Stop()
	defer queueConsumedConsumer.Stop()

	//
	// Create a simple canvas with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "noop"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	support.CreateQueueItem(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
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
		worker1 := NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)
		logger := log.NewEntry(log.New())
		results <- worker1.LockAndProcessNode(logger, *node, time.Now())
	}()

	go func() {
		worker2 := NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)
		logger := log.NewEntry(log.New())
		results <- worker2.LockAndProcessNode(logger, *node, time.Now())
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
	executions, err := models.ListNodeExecutions(canvas.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	assert.Len(t, executions, 1, "Only one execution should be created, not two")

	//
	// Verify the queue item was deleted.
	//
	queueItems, err := models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, queueItems, 0, "Queue item should be deleted")

	assert.True(t, executionConsumer.HasReceivedMessage())
	assert.True(t, queueConsumedConsumer.HasReceivedMessage())
}

func Test__NodeQueueWorker_ConfigurationBuildFailure(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)
	logger := log.NewEntry(log.New())

	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionFinishedRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a trigger and a component node.
	// The component node will have a configuration with an invalid expression.
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, nodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	support.CreateQueueItem(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)

	//
	// Verify initial state - node should be ready and queue should have 1 item.
	//
	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateReady, node.State)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)

	//
	// Process the node - this should handle the configuration build failure:
	// - Failed execution should be created
	// - Node state should remain ready (not updated to processing)
	// - Queue item should be deleted
	//
	err = worker.LockAndProcessNode(logger, *node, time.Now())
	require.NoError(t, err)

	// Verify execution was created with finished state and failed result
	executions, err := models.ListNodeExecutions(canvas.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, executions[0].State)
	assert.Equal(t, models.CanvasNodeExecutionResultFailed, executions[0].Result)
	assert.Equal(t, models.CanvasNodeExecutionResultReasonError, executions[0].ResultReason)
	assert.Contains(t, executions[0].ResultMessage, "field")
	assert.Equal(t, rootEvent.ID, executions[0].EventID)
	assert.Equal(t, rootEvent.ID, executions[0].RootEventID)
	assert.Equal(t, nodes[1].Configuration, executions[0].Configuration)

	// Verify node state is still ready (not updated to processing)
	node, err = models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateReady, node.State)

	// Verify queue item was deleted
	queueItems, err = models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, queueItems, 0)

	// Verify execution message was published
	assert.True(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeQueueWorker_ProcessesNextQueueItemOnExecutionFinished(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)
	logger := log.NewEntry(log.New())

	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNode, Type: models.NodeTypeTrigger},
			{NodeID: componentNode, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	oldEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	newEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)

	oldTime := time.Now().Add(-5 * time.Minute)
	newTime := time.Now()
	oldQueueItem := models.CanvasNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		RootEventID: oldEvent.ID,
		EventID:     oldEvent.ID,
		CreatedAt:   &oldTime,
	}
	newQueueItem := models.CanvasNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		RootEventID: newEvent.ID,
		EventID:     newEvent.ID,
		CreatedAt:   &newTime,
	}
	require.NoError(t, database.Conn().Create(&oldQueueItem).Error)
	require.NoError(t, database.Conn().Create(&newQueueItem).Error)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)

	//
	// Process the first queue item while the node is ready.
	//
	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	executions, err := models.ListNodeExecutions(canvas.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	require.Equal(t, oldEvent.ID, executions[0].EventID)
	require.Equal(t, models.CanvasNodeExecutionStatePending, executions[0].State)

	node, err = models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateProcessing, node.State)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)
	assert.Equal(t, newEvent.ID, queueItems[0].EventID)

	//
	// Finish the execution so the node becomes ready again.
	//
	execution := executions[0]
	require.NoError(t, execution.Start())
	_, err = execution.Pass(nil)
	require.NoError(t, err)

	node, err = models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateReady, node.State)

	//
	// Simulate the execution.finished RabbitMQ message emitted by the executor.
	//
	finishedMessage, err := proto.Marshal(&pb.CanvasNodeExecutionMessage{
		Id:       execution.ID.String(),
		CanvasId: canvas.ID.String(),
		NodeId:   componentNode,
	})
	require.NoError(t, err)

	require.NoError(t, worker.ConsumeExecutionFinished(tackle.NewFakeDelivery(finishedMessage)))

	executions, err = models.ListNodeExecutions(canvas.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 2)
	assert.Equal(t, newEvent.ID, executions[0].EventID)
	assert.Equal(t, models.CanvasNodeExecutionStatePending, executions[0].State)

	queueItems, err = models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)
}

func Test__NodeQueueWorker_DiscardsQueueItemForFinishedRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)
	logger := log.NewEntry(log.New())

	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)

	//
	// Create the run for the root event and finalize it as cancelled, simulating a
	// queue item that was created just before the run was cancelled.
	//
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), rootEvent)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, database.Conn().Model(run).Updates(map[string]any{
		"state":        models.CanvasRunStateFinished,
		"result":       models.CanvasRunResultCancelled,
		"cancelled_at": &now,
		"finished_at":  &now,
		"updated_at":   &now,
	}).Error)

	queueItem := support.CreateQueueItem(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)
	queueItem.RunID = run.ID
	require.NoError(t, database.Conn().Save(queueItem).Error)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNode)
	require.NoError(t, err)

	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	//
	// No execution should be created, and the stale queue item should be discarded.
	//
	executions, err := models.ListNodeExecutions(canvas.ID, componentNode, nil, nil, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, executions)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)
}
