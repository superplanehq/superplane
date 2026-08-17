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
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

//
// Acceptance tests for the queueing-and-parallelism POC
// (docs/prd/queueing-and-parallelism.md). Workers are driven manually:
// LockAndProcessNode dispatches queue items, executions are passed/failed
// by hand, and the event router routes emitted events.
//

func queueWorkerForTest(r *support.ResourceRegistry) *NodeQueueWorker {
	amqpURL, _ := config.RabbitMQURL()
	return NewNodeQueueWorker(r.Registry, r.GitProvider, amqpURL)
}

func processQueueNode(t *testing.T, worker *NodeQueueWorker, canvasID uuid.UUID, nodeID string) {
	t.Helper()
	node, err := models.FindCanvasNode(database.Conn(), canvasID, nodeID)
	require.NoError(t, err)
	require.NoError(t, worker.LockAndProcessNode(log.NewEntry(log.New()), *node, time.Now()))
}

func listNodeExecutionsForTest(t *testing.T, canvasID uuid.UUID, nodeID string) []models.CanvasNodeExecution {
	t.Helper()
	executions, err := models.ListNodeExecutions(database.Conn(), canvasID, nodeID, nil, nil, 100, nil)
	require.NoError(t, err)
	return executions
}

func listActiveNodeExecutionsForTest(t *testing.T, canvasID uuid.UUID, nodeID string) []models.CanvasNodeExecution {
	t.Helper()
	executions, err := models.ListActiveNodeExecutions(database.Conn(), canvasID, nodeID)
	require.NoError(t, err)
	return executions
}

func listQueueItemsForTest(t *testing.T, canvasID uuid.UUID, nodeID string) []models.CanvasNodeQueueItem {
	t.Helper()
	items, err := models.ListNodeQueueItems(database.Conn(), canvasID, nodeID, 100, nil)
	require.NoError(t, err)
	return items
}

func createQueueItemAt(t *testing.T, canvasID uuid.UUID, nodeID string, event *models.CanvasEvent, createdAt time.Time) *models.CanvasNodeQueueItem {
	t.Helper()
	item := models.CanvasNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  canvasID,
		NodeID:      nodeID,
		RootEventID: event.ID,
		EventID:     event.ID,
		CreatedAt:   &createdAt,
	}
	require.NoError(t, database.Conn().Create(&item).Error)
	return &item
}

func concurrencyMax(limit int) *int {
	return &limit
}

func triggerAndComponent(componentNodeID string, concurrency *models.ConcurrencySpec) []models.CanvasNode {
	return []models.CanvasNode{
		{
			NodeID: "trigger-1",
			Type:   models.NodeTypeTrigger,
			Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
		},
		{
			NodeID:      componentNodeID,
			Type:        models.NodeTypeComponent,
			Ref:         datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			Concurrency: models.ConcurrencySpecColumn(concurrency),
		},
	}
}

func triggerToComponentEdge(componentNodeID string) []models.Edge {
	return []models.Edge{
		{SourceID: "trigger-1", TargetID: componentNodeID, Channel: "default"},
	}
}

// Scenario 1: a node with concurrency { max: 3 } and five queued
// inputs runs exactly three executions concurrently, FIFO; as one
// finishes, the next dispatches.
func Test__Queueing_NodeQueueRunsUpToConcurrencyMax(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := queueWorkerForTest(r)
	componentNode := "agent-node"

	canvas, _ := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		triggerAndComponent(componentNode, &models.ConcurrencySpec{Max: concurrencyMax(3)}),
		triggerToComponentEdge(componentNode),
	)

	base := time.Now().Add(-10 * time.Minute)
	events := make([]*models.CanvasEvent, 5)
	for i := range events {
		events[i] = support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
		createQueueItemAt(t, canvas.ID, componentNode, events[i], base.Add(time.Duration(i)*time.Minute))
	}

	//
	// One pass dispatches exactly three executions, in FIFO order.
	//
	processQueueNode(t, worker, canvas.ID, componentNode)

	executions := listNodeExecutionsForTest(t, canvas.ID, componentNode)
	require.Len(t, executions, 3)
	dispatchedEvents := []uuid.UUID{executions[0].EventID, executions[1].EventID, executions[2].EventID}
	assert.ElementsMatch(t, dispatchedEvents, []uuid.UUID{events[0].ID, events[1].ID, events[2].ID})
	for _, execution := range executions {
		require.NotNil(t, execution.QueueName)
		assert.Equal(t, componentNode, *execution.QueueName)
	}

	assert.Len(t, listQueueItemsForTest(t, canvas.ID, componentNode), 2)

	//
	// Re-running while at capacity dispatches nothing.
	//
	processQueueNode(t, worker, canvas.ID, componentNode)
	assert.Len(t, listNodeExecutionsForTest(t, canvas.ID, componentNode), 3)

	//
	// Finishing one execution frees one slot: the fourth input dispatches.
	//
	first := findExecutionByEvent(t, canvas.ID, componentNode, events[0].ID)
	require.NoError(t, first.Start())
	_, err := first.Pass(nil)
	require.NoError(t, err)

	processQueueNode(t, worker, canvas.ID, componentNode)

	active := listActiveNodeExecutionsForTest(t, canvas.ID, componentNode)
	require.Len(t, active, 3)
	assert.NotNil(t, findExecutionByEvent(t, canvas.ID, componentNode, events[3].ID))
	assert.Len(t, listQueueItemsForTest(t, canvas.ID, componentNode), 1)
}

// Scenario 2: a node with no concurrency field behaves as today: one execution
// at a time, FIFO, next item only after the previous execution finishes.
func Test__Queueing_ImplicitQueueSerializesNode(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := queueWorkerForTest(r)
	componentNode := "serial-node"

	canvas, _ := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		triggerAndComponent(componentNode, nil),
		triggerToComponentEdge(componentNode),
	)

	base := time.Now().Add(-10 * time.Minute)
	firstEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	secondEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	createQueueItemAt(t, canvas.ID, componentNode, firstEvent, base)
	createQueueItemAt(t, canvas.ID, componentNode, secondEvent, base.Add(time.Minute))

	//
	// First pass dispatches only the oldest item.
	//
	processQueueNode(t, worker, canvas.ID, componentNode)
	executions := listNodeExecutionsForTest(t, canvas.ID, componentNode)
	require.Len(t, executions, 1)
	assert.Equal(t, firstEvent.ID, executions[0].EventID)
	require.NotNil(t, executions[0].QueueName)
	assert.Equal(t, componentNode, *executions[0].QueueName)

	//
	// While the execution is active, nothing else dispatches.
	//
	processQueueNode(t, worker, canvas.ID, componentNode)
	assert.Len(t, listNodeExecutionsForTest(t, canvas.ID, componentNode), 1)

	//
	// After it finishes, the second item dispatches.
	//
	execution := executions[0]
	require.NoError(t, execution.Start())
	_, err := execution.Pass(nil)
	require.NoError(t, err)

	processQueueNode(t, worker, canvas.ID, componentNode)
	executions = listNodeExecutionsForTest(t, canvas.ID, componentNode)
	require.Len(t, executions, 2)
	assert.NotNil(t, findExecutionByEvent(t, canvas.ID, componentNode, secondEvent.ID))
}

// Scenario 3: monorepo case. A node with queue key "ci-{{ root().branch }}"
// serializes per branch but runs different branches in parallel.
func Test__Queueing_ExpressionKeyPartitionsByBranch(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := queueWorkerForTest(r)
	componentNode := "ci-node"

	canvas, _ := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		triggerAndComponent(componentNode, &models.ConcurrencySpec{Key: "ci-{{ root().branch }}"}),
		triggerToComponentEdge(componentNode),
	)

	base := time.Now().Add(-10 * time.Minute)
	authEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "trigger-1", "default", nil, map[string]any{"branch": "feature-auth"})
	cartEvent1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "trigger-1", "default", nil, map[string]any{"branch": "feature-cart"})
	cartEvent2 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "trigger-1", "default", nil, map[string]any{"branch": "feature-cart"})

	createQueueItemAt(t, canvas.ID, componentNode, authEvent, base)
	createQueueItemAt(t, canvas.ID, componentNode, cartEvent1, base.Add(time.Minute))
	createQueueItemAt(t, canvas.ID, componentNode, cartEvent2, base.Add(2*time.Minute))

	//
	// One pass: both branches dispatch in parallel; the second push to
	// feature-cart waits behind the first one.
	//
	processQueueNode(t, worker, canvas.ID, componentNode)

	executions := listNodeExecutionsForTest(t, canvas.ID, componentNode)
	require.Len(t, executions, 2)

	authExecution := findExecutionByEvent(t, canvas.ID, componentNode, authEvent.ID)
	require.NotNil(t, authExecution)
	assert.Equal(t, "ci-feature-auth", *authExecution.QueueName)

	cartExecution := findExecutionByEvent(t, canvas.ID, componentNode, cartEvent1.ID)
	require.NotNil(t, cartExecution)
	assert.Equal(t, "ci-feature-cart", *cartExecution.QueueName)

	items := listQueueItemsForTest(t, canvas.ID, componentNode)
	require.Len(t, items, 1)
	assert.Equal(t, cartEvent2.ID, items[0].EventID)
	require.NotNil(t, items[0].QueueName)
	assert.Equal(t, "ci-feature-cart", *items[0].QueueName)

	//
	// When the first feature-cart run finishes, the second dispatches.
	//
	require.NoError(t, cartExecution.Start())
	_, err := cartExecution.Pass(nil)
	require.NoError(t, err)

	processQueueNode(t, worker, canvas.ID, componentNode)
	assert.NotNil(t, findExecutionByEvent(t, canvas.ID, componentNode, cartEvent2.ID))
	assert.Empty(t, listQueueItemsForTest(t, canvas.ID, componentNode))
}

// Scenario 4: docs-deploy case. A queue with autoCancel: queued keeps only
// the newest waiting item; older waiting items and their runs are recorded
// as superseded, not failed.
func Test__Queueing_AutoCancelQueuedSupersedesOlderItems(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := queueWorkerForTest(r)
	componentNode := "docs-deploy-node"

	canvas, _ := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		triggerAndComponent(componentNode, &models.ConcurrencySpec{AutoCancel: models.QueueAutoCancelQueued}),
		triggerToComponentEdge(componentNode),
	)

	base := time.Now().Add(-10 * time.Minute)
	runningEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	createQueueItemAt(t, canvas.ID, componentNode, runningEvent, base)

	processQueueNode(t, worker, canvas.ID, componentNode)
	runningExecution := findExecutionByEvent(t, canvas.ID, componentNode, runningEvent.ID)
	require.NotNil(t, runningExecution)
	require.NoError(t, runningExecution.Start())

	//
	// Three more items arrive while the execution is running.
	//
	waitingEvents := make([]*models.CanvasEvent, 3)
	waitingItems := make([]*models.CanvasNodeQueueItem, 3)
	for i := range waitingEvents {
		waitingEvents[i] = support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
		waitingItems[i] = createQueueItemAt(t, canvas.ID, componentNode, waitingEvents[i], base.Add(time.Duration(i+1)*time.Minute))

		//
		// The router marks root events as routed before queue items are
		// worked on; superseding only finishes a run with no pending events.
		//
		require.NoError(t, database.Conn().Model(waitingEvents[i]).Update("state", models.CanvasEventStateRouted).Error)
	}

	//
	// The pass supersedes the two older waiting items; the newest waits for
	// capacity. The running execution is untouched.
	//
	processQueueNode(t, worker, canvas.ID, componentNode)

	items := listQueueItemsForTest(t, canvas.ID, componentNode)
	require.Len(t, items, 1)
	assert.Equal(t, waitingEvents[2].ID, items[0].EventID)

	assert.Len(t, listNodeExecutionsForTest(t, canvas.ID, componentNode), 1)

	for i := 0; i < 2; i++ {
		run, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, waitingItems[i].RunID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasRunStateFinished, run.State)
		assert.Equal(t, models.CanvasRunResultSuperseded, run.Result)
	}

	//
	// When the running execution finishes, the newest item dispatches.
	//
	_, err := runningExecution.Pass(nil)
	require.NoError(t, err)

	processQueueNode(t, worker, canvas.ID, componentNode)
	assert.NotNil(t, findExecutionByEvent(t, canvas.ID, componentNode, waitingEvents[2].ID))
	assert.Empty(t, listQueueItemsForTest(t, canvas.ID, componentNode))
}

// Scenario 5: autoCancel: running. A new item cancels the queue's in-flight
// execution and dispatches after the cancellation completes. Other nodes'
// queues are unaffected.
func Test__Queueing_AutoCancelRunningCancelsInFlightExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := queueWorkerForTest(r)
	apiNode := "api-node"
	otherNode := "other-node"

	canvas, _ := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger-1",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID:      apiNode,
				Type:        models.NodeTypeComponent,
				Ref:         datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
				Concurrency: models.ConcurrencySpecColumn(&models.ConcurrencySpec{AutoCancel: models.QueueAutoCancelRunning}),
			},
			{
				NodeID: otherNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: "trigger-1", TargetID: apiNode, Channel: "default"},
			{SourceID: "trigger-1", TargetID: otherNode, Channel: "default"},
		},
	)

	base := time.Now().Add(-10 * time.Minute)

	//
	// An execution runs in the "other" queue throughout; it must stay untouched.
	//
	otherEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	createQueueItemAt(t, canvas.ID, otherNode, otherEvent, base)
	processQueueNode(t, worker, canvas.ID, otherNode)
	otherExecution := findExecutionByEvent(t, canvas.ID, otherNode, otherEvent.ID)
	require.NotNil(t, otherExecution)
	require.NoError(t, otherExecution.Start())

	firstEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	createQueueItemAt(t, canvas.ID, apiNode, firstEvent, base)

	processQueueNode(t, worker, canvas.ID, apiNode)
	firstExecution := findExecutionByEvent(t, canvas.ID, apiNode, firstEvent.ID)
	require.NotNil(t, firstExecution)
	require.NoError(t, firstExecution.Start())

	//
	// A new item arrives: the in-flight execution moves to cancelling and
	// the new item stays queued until the cancellation completes.
	//
	secondEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	createQueueItemAt(t, canvas.ID, apiNode, secondEvent, base.Add(time.Minute))

	processQueueNode(t, worker, canvas.ID, apiNode)

	firstExecution = findExecutionByEvent(t, canvas.ID, apiNode, firstEvent.ID)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, firstExecution.State)
	assert.Len(t, listQueueItemsForTest(t, canvas.ID, apiNode), 1)
	assert.Len(t, listNodeExecutionsForTest(t, canvas.ID, apiNode), 1)

	//
	// Simulate the executor completing the cancellation. The next pass
	// dispatches the new item.
	//
	require.NoError(t, firstExecution.Cancel(nil))

	processQueueNode(t, worker, canvas.ID, apiNode)
	assert.NotNil(t, findExecutionByEvent(t, canvas.ID, apiNode, secondEvent.ID))
	assert.Empty(t, listQueueItemsForTest(t, canvas.ID, apiNode))

	//
	// The "other" queue was unaffected.
	//
	otherExecution = findExecutionByEvent(t, canvas.ID, otherNode, otherEvent.ID)
	assert.Equal(t, models.CanvasNodeExecutionStateStarted, otherExecution.State)
}

// Scenario 13: deploy → tests gating. A group over [deploy, test] admits
// one run at a time (default max 1); the slot is released when
// the run's work leaves the group (section-end), not when the run
// finishes, and a run failing at deploy releases the slot without running
// tests.
func Test__Queueing_GroupQueueGatesSectionAcrossRuns(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	worker := queueWorkerForTest(r)
	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())

	deployNode := "deploy"
	testNode := "test"
	restNode := "rest"

	canvas, _ := support.CreateCanvasWithNodeGroups(
		t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger-1",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: deployNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: testNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: restNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: "trigger-1", TargetID: deployNode, Channel: "default"},
			{SourceID: deployNode, TargetID: testNode, Channel: "default"},
			{SourceID: testNode, TargetID: restNode, Channel: "default"},
		},
		[]models.NodeGroup{
			{ID: "staging-group", Nodes: []string{deployNode, testNode}},
		},
	)

	routeEvent := func(event *models.CanvasEvent) {
		t.Helper()
		require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))
	}

	passExecution := func(execution *models.CanvasNodeExecution) {
		t.Helper()
		require.NoError(t, execution.Start())
		events, err := execution.Pass(map[string][]any{"default": {map[string]any{"ok": true}}})
		require.NoError(t, err)
		for i := range events {
			routeEvent(&events[i])
		}
	}

	stagingSlots := func() []models.CanvasQueueSlot {
		t.Helper()
		var slots []models.CanvasQueueSlot
		require.NoError(t, database.Conn().
			Where("workflow_id = ?", canvas.ID).
			Where("group_id = ?", "staging-group").
			Find(&slots).Error)
		return slots
	}

	//
	// Two runs arrive at deploy. Run 1 acquires the staging slot; run 2 waits.
	//
	rootEvent1 := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	routeEvent(rootEvent1)
	rootEvent2 := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	routeEvent(rootEvent2)

	require.Len(t, listQueueItemsForTest(t, canvas.ID, deployNode), 2)

	processQueueNode(t, worker, canvas.ID, deployNode)

	deployExecutions := listNodeExecutionsForTest(t, canvas.ID, deployNode)
	require.Len(t, deployExecutions, 1)
	deployExecution1 := deployExecutions[0]
	require.Equal(t, rootEvent1.ID, deployExecution1.EventID)

	slots := stagingSlots()
	require.Len(t, slots, 1)
	assert.Equal(t, deployExecution1.RunID, slots[0].RunID)
	require.Len(t, listQueueItemsForTest(t, canvas.ID, deployNode), 1)

	//
	// Run 1's deploy finishes and its event routes to test. The run still
	// has work inside the group, so the slot is kept and run 2 stays blocked.
	//
	passExecution(&deployExecution1)
	require.Len(t, stagingSlots(), 1)

	processQueueNode(t, worker, canvas.ID, deployNode)
	assert.Len(t, listNodeExecutionsForTest(t, canvas.ID, deployNode), 1)

	//
	// Run 1's test dispatches inside the same group (same slot) and passes.
	// Its output leaves the group, so the slot is released — even though
	// run 1's rest node has not executed yet.
	//
	processQueueNode(t, worker, canvas.ID, testNode)
	testExecutions := listNodeExecutionsForTest(t, canvas.ID, testNode)
	require.Len(t, testExecutions, 1)

	passExecution(&testExecutions[0])

	assert.Empty(t, stagingSlots())
	require.Len(t, listQueueItemsForTest(t, canvas.ID, restNode), 1, "run 1 still has work outside the group")

	//
	// Run 2's deploy dispatches now and takes the slot.
	//
	processQueueNode(t, worker, canvas.ID, deployNode)

	deployExecutions = listNodeExecutionsForTest(t, canvas.ID, deployNode)
	require.Len(t, deployExecutions, 2)
	deployExecution2 := findExecutionByEvent(t, canvas.ID, deployNode, rootEvent2.ID)
	require.NotNil(t, deployExecution2)

	slots = stagingSlots()
	require.Len(t, slots, 1)
	assert.Equal(t, deployExecution2.RunID, slots[0].RunID)

	//
	// A run that fails at deploy releases the slot without running tests.
	//
	rootEvent3 := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	routeEvent(rootEvent3)

	require.NoError(t, deployExecution2.Start())
	_, err := deployExecution2.FailInTransaction(database.Conn(), "error", "deploy failed")
	require.NoError(t, err)

	assert.Empty(t, stagingSlots())
	assert.Empty(t, listQueueItemsForTest(t, canvas.ID, testNode), "failed deploy does not queue tests")

	processQueueNode(t, worker, canvas.ID, deployNode)
	deployExecution3 := findExecutionByEvent(t, canvas.ID, deployNode, rootEvent3.ID)
	require.NotNil(t, deployExecution3)

	slots = stagingSlots()
	require.Len(t, slots, 1)
	assert.Equal(t, deployExecution3.RunID, slots[0].RunID)
}

// Self-managed components must never be capacity-gated. A merge keeps its
// execution open while waiting for the remaining sources; counting that
// execution against the implicit limit of 1 would block the second
// source's queue item forever and deadlock the merge.
func Test__Queueing_SelfManagedMergeIsNotCapacityGated(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	worker := queueWorkerForTest(r)
	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())

	mergeNode := "merge-node"

	canvas, _ := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger-1",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: "branch-a",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "branch-b",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: mergeNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}}),
			},
		},
		[]models.Edge{
			{SourceID: "trigger-1", TargetID: "branch-a", Channel: "default"},
			{SourceID: "trigger-1", TargetID: "branch-b", Channel: "default"},
			{SourceID: "branch-a", TargetID: mergeNode, Channel: "default"},
			{SourceID: "branch-b", TargetID: mergeNode, Channel: "default"},
		},
	)

	routeEvent := func(event *models.CanvasEvent) {
		t.Helper()
		require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))
	}

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	routeEvent(rootEvent)

	//
	// Both branches run and route their outputs into the merge node.
	//
	for _, branch := range []string{"branch-a", "branch-b"} {
		processQueueNode(t, worker, canvas.ID, branch)
		executions := listNodeExecutionsForTest(t, canvas.ID, branch)
		require.Len(t, executions, 1)
		require.NoError(t, executions[0].Start())
		events, err := executions[0].Pass(map[string][]any{"default": {map[string]any{"ok": true}}})
		require.NoError(t, err)
		for i := range events {
			routeEvent(&events[i])
		}
	}

	require.Len(t, listQueueItemsForTest(t, canvas.ID, mergeNode), 2)

	//
	// One pass consumes both source items: the first opens the merge
	// execution, the second completes it. Capacity-gating the second item
	// against the open execution would deadlock here.
	//
	processQueueNode(t, worker, canvas.ID, mergeNode)

	assert.Empty(t, listQueueItemsForTest(t, canvas.ID, mergeNode))

	executions := listNodeExecutionsForTest(t, canvas.ID, mergeNode)
	require.Len(t, executions, 1)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, executions[0].State)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, executions[0].Result)
}

func findExecutionByEvent(t *testing.T, canvasID uuid.UUID, nodeID string, eventID uuid.UUID) *models.CanvasNodeExecution {
	t.Helper()
	executions := listNodeExecutionsForTest(t, canvasID, nodeID)
	for i := range executions {
		if executions[i].EventID == eventID {
			return &executions[i]
		}
	}
	return nil
}
