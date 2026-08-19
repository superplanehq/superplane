package workers

import (
	"slices"
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
// Acceptance tests for node concurrency (per-node queues with a
// configurable max and key expressions). Workers are driven manually:
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

// createResolvedQueueItemAt creates a queue item whose queue name is
// already resolved, as if a previous dispatch pass touched it.
func createResolvedQueueItemAt(t *testing.T, canvasID uuid.UUID, nodeID string, event *models.CanvasEvent, queueName string, createdAt time.Time) *models.CanvasNodeQueueItem {
	t.Helper()
	item := createQueueItemAt(t, canvasID, nodeID, event, createdAt)
	require.NoError(t, database.Conn().Model(item).Update("queue_name", queueName).Error)
	return item
}

func concurrencyMax(limit int) *int {
	return &limit
}

func triggerAndComponent(componentNodeID string, concurrency *models.ConcurrencySpec) []models.CanvasNode {
	component := models.CanvasNode{
		NodeID: componentNodeID,
		Type:   models.NodeTypeComponent,
		Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
	}
	component.SetConcurrencySpec(concurrency)

	return []models.CanvasNode{
		{
			NodeID: "trigger-1",
			Type:   models.NodeTypeTrigger,
			Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
		},
		component,
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

	runBranch := func(branch string) {
		t.Helper()
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

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
	routeEvent(rootEvent)

	//
	// The first branch's item opens the merge execution, which stays open
	// waiting for the second input.
	//
	runBranch("branch-a")
	processQueueNode(t, worker, canvas.ID, mergeNode)
	require.Len(t, listActiveNodeExecutionsForTest(t, canvas.ID, mergeNode), 1)

	//
	// The second input arrives while the execution is open. Self-managed
	// items keep a NULL queue name, which is what keeps the node in the
	// poll set: capacity-gating this item against the open execution
	// (implicit max 1) would deadlock the merge.
	//
	runBranch("branch-b")
	items := listQueueItemsForTest(t, canvas.ID, mergeNode)
	require.Len(t, items, 1)
	assert.Nil(t, items[0].QueueName)
	assert.True(t, pollReturnsNode(t, canvas.ID, mergeNode))

	//
	// The next pass hands the item to the component, completing the merge.
	// The execution never entered a named queue.
	//
	processQueueNode(t, worker, canvas.ID, mergeNode)

	assert.Empty(t, listQueueItemsForTest(t, canvas.ID, mergeNode))

	executions := listNodeExecutionsForTest(t, canvas.ID, mergeNode)
	require.Len(t, executions, 1)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, executions[0].State)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, executions[0].Result)
	assert.Nil(t, executions[0].QueueName)
}

// A loop honors concurrency.max for its parallel sessions even within a
// single dispatch pass. Sessions created earlier in the pass are still
// pending (the executor has not started them), so the slot count must
// include pending executions or every backlog item would open a session.
func Test__Queueing_LoopSessionsRespectMaxWithinOnePass(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := queueWorkerForTest(r)
	loopNode := "loop-node"

	canvas, _ := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger-1",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: loopNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "loop"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"untilExpression": `$["Checker"].ready == true`,
				}),
			},
		},
		triggerToComponentEdge(loopNode),
	)

	base := time.Now().Add(-10 * time.Minute)
	for i := 0; i < 2; i++ {
		event := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
		createQueueItemAt(t, canvas.ID, loopNode, event, base.Add(time.Duration(i)*time.Minute))
	}

	//
	// One pass: the first item opens a session; the second is deferred
	// because the still-pending session holds the only slot (default
	// max 1).
	//
	processQueueNode(t, worker, canvas.ID, loopNode)

	assert.Len(t, listActiveNodeExecutionsForTest(t, canvas.ID, loopNode), 1)
	assert.Len(t, listQueueItemsForTest(t, canvas.ID, loopNode), 1)
}

// A deep backlog on one busy queue must not starve other queues. The
// dispatch pass scans at most queueItemScanLimit items; if items waiting
// on a full queue counted against that window, a busy key with a backlog
// deeper than the limit would keep items of other keys out of every pass.
func Test__Queueing_DeepBlockedBacklogDoesNotStarveOtherQueues(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := queueWorkerForTest(r)
	componentNode := "ci-node"

	canvas, _ := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		triggerAndComponent(componentNode, &models.ConcurrencySpec{Key: "ci-{{ root().branch }}"}),
		triggerToComponentEdge(componentNode),
	)

	base := time.Now().Add(-10 * time.Hour)

	//
	// The busy branch fills its only slot (default max 1)...
	//
	busyEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "trigger-1", "default", nil, map[string]any{"branch": "busy"})
	createQueueItemAt(t, canvas.ID, componentNode, busyEvent, base)
	processQueueNode(t, worker, canvas.ID, componentNode)
	require.Len(t, listActiveNodeExecutionsForTest(t, canvas.ID, componentNode), 1)

	//
	// ...and accumulates a resolved backlog deeper than the scan window.
	//
	for i := 0; i < queueItemScanLimit; i++ {
		createResolvedQueueItemAt(t, canvas.ID, componentNode, busyEvent, "ci-busy", base.Add(time.Duration(i+1)*time.Minute))
	}

	//
	// An item for another branch arrives behind the whole busy backlog.
	//
	idleEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "trigger-1", "default", nil, map[string]any{"branch": "idle"})
	createQueueItemAt(t, canvas.ID, componentNode, idleEvent, base.Add(time.Duration(queueItemScanLimit+1)*time.Minute))

	//
	// One pass dispatches it: blocked busy items do not consume the scan
	// window. The busy backlog itself stays queued.
	//
	processQueueNode(t, worker, canvas.ID, componentNode)

	idleExecution := findExecutionByEvent(t, canvas.ID, componentNode, idleEvent.ID)
	require.NotNil(t, idleExecution)
	require.NotNil(t, idleExecution.QueueName)
	assert.Equal(t, "ci-idle", *idleExecution.QueueName)

	items, err := models.ListNodeQueueItems(database.Conn(), canvas.ID, componentNode, queueItemScanLimit+10, nil)
	require.NoError(t, err)
	assert.Len(t, items, queueItemScanLimit)
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

func pollReturnsNode(t *testing.T, canvasID uuid.UUID, nodeID string) bool {
	t.Helper()
	nodes, err := models.ListCanvasNodesReady(database.Conn())
	require.NoError(t, err)
	return containsCanvasNode(nodes, canvasID, nodeID)
}

func containsCanvasNode(nodes []models.CanvasNode, canvasID uuid.UUID, nodeID string) bool {
	return slices.ContainsFunc(nodes, func(n models.CanvasNode) bool {
		return n.WorkflowID == canvasID && n.NodeID == nodeID
	})
}

// The safety-net poll only returns nodes a dispatch pass can make progress
// on. A node whose whole backlog waits on full queues is skipped until an
// execution finishes; items without a queue name (unresolved, or
// self-managed components' items, which never get one) keep a node in the
// poll set.
func Test__Queueing_PollSkipsNodesWithoutActionableBacklog(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := queueWorkerForTest(r)
	componentNode := "poll-node"

	canvas, _ := support.CreateCanvas(
		t, r.Organization.ID, r.User,
		triggerAndComponent(componentNode, nil),
		triggerToComponentEdge(componentNode),
	)

	base := time.Now().Add(-10 * time.Minute)
	events := make([]*models.CanvasEvent, 2)
	for i := range events {
		events[i] = support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
		createQueueItemAt(t, canvas.ID, componentNode, events[i], base.Add(time.Duration(i)*time.Minute))
	}

	//
	// An unresolved backlog is always actionable: queue names are only
	// known once the worker touches the items.
	//
	assert.True(t, pollReturnsNode(t, canvas.ID, componentNode))

	//
	// One pass dispatches the first item (default max 1) and resolves the
	// queue name of the second. The backlog now waits on a full queue, so
	// the poll skips the node.
	//
	processQueueNode(t, worker, canvas.ID, componentNode)
	require.Len(t, listActiveNodeExecutionsForTest(t, canvas.ID, componentNode), 1)
	items := listQueueItemsForTest(t, canvas.ID, componentNode)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].QueueName)

	assert.False(t, pollReturnsNode(t, canvas.ID, componentNode))

	//
	// Finishing the execution frees the queue: the node is actionable again.
	//
	execution := findExecutionByEvent(t, canvas.ID, componentNode, events[0].ID)
	require.NoError(t, execution.Start())
	_, err := execution.Pass(nil)
	require.NoError(t, err)

	assert.True(t, pollReturnsNode(t, canvas.ID, componentNode))
}
