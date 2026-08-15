package workers

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers/onerror"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func newTestExecutorForReplay(r *support.ResourceRegistry) *NodeExecutor {
	return NewNodeExecutor(
		r.Encryptor,
		r.Registry,
		r.GitProvider,
		support.NewOIDCProvider(),
		"http://localhost",
		"http://localhost",
		"",
		r.AuthService,
	)
}

// Given a component node and a payload P, replaying it creates a run
// flagged as a replay, an execution referencing the source execution, the
// node executes, and ctx.Data equals P.
func Test__Replay_CreatesFlaggedRunAndExecutionWithPayload(t *testing.T) {
	componentName := "replay-dod6-" + uuid.New().String()

	var capturedData any
	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name: componentName,
		ExecuteFunc: func(ctx core.ExecutionContext) error {
			capturedData = ctx.Data
			return ctx.ExecutionState.Pass()
		},
	}))

	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
			{NodeID: componentNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}})},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	//
	// Original execution being replayed.
	//
	originalEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, map[string]any{"v": "original"})
	originalExecution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, originalEvent.ID, originalEvent.ID)
	_, err := originalExecution.Pass(map[string][]any{})
	require.NoError(t, err, "the original execution is history: it is already finished by the time it is replayed")

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNodeID)
	require.NoError(t, err)

	payload := map[string]any{"v": "replay-payload"}

	var run *models.CanvasRun
	var queueItems []*models.CanvasNodeQueueItem
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		run, queueItems, txErr = models.CreateMultiInputNodeReplay(tx, node, &originalExecution.ID, []models.ReplayInput{{
			Payload:           payload,
			SourceExecutionID: &originalExecution.ID,
		}})
		return txErr
	})
	require.NoError(t, err)
	require.True(t, run.IsReplay)
	require.NotNil(t, run.ReplaySourceExecutionID)
	assert.Equal(t, originalExecution.ID, *run.ReplaySourceExecutionID)

	//
	// Enter through the node queue - LockAndProcessNode picks up the queue
	// item exactly like it does for a normal root event.
	//
	logger := log.NewEntry(log.New())
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, "")
	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	_, err = models.FindNodeQueueItem(canvas.ID, queueItems[0].ID)
	assert.Error(t, err, "queue item should have been consumed")

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, componentNodeID, []string{models.CanvasNodeExecutionStatePending}, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	require.Equal(t, run.ID, executions[0].RunID)

	executor := newTestExecutorForReplay(r)
	require.NoError(t, executor.LockAndProcessNodeExecution(executions[0].ID))

	assert.Equal(t, payload, capturedData, "ctx.Data must equal the replay payload P, not the original execution's input")
}

// A replay whose configuration cannot be built fails through
// handleNodeConfigurationError, which dispatches to On Error nodes without
// going through ExecutionStateContext.Fail. The same canvas dispatches for a
// normal run, so the absence below is isolation, not a dead dispatch path.
func Test__Replay_ConfigurationBuildFailureDoesNotNotifyOnErrorNodes(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "component-1"
	onErrorNodeID := "onerror-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
			{
				NodeID:        componentNodeID,
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
				Configuration: datatypes.NewJSONType(map[string]any{"field": "{{ $[\"nonexistent-node\"].data }}"}),
			},
			{NodeID: onErrorNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: onerror.TriggerName}})},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	logger := log.NewEntry(log.New())
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, "")

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNodeID)
	require.NoError(t, err)

	//
	// Control: an ordinary run hitting the same configuration error does reach
	// the On Error node.
	//
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNodeID, "default", nil)
	support.CreateQueueItem(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID)
	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	onErrorEvents, err := models.ListCanvasEvents(database.Conn(), canvas.ID, onErrorNodeID, 10, nil)
	require.NoError(t, err)
	require.Len(t, onErrorEvents, 1, "a non-replay configuration failure must notify the On Error node")

	var run *models.CanvasRun
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		run, _, txErr = models.CreateMultiInputNodeReplay(tx, node, nil, []models.ReplayInput{{Payload: map[string]any{"a": 1}}})
		return txErr
	})
	require.NoError(t, err)
	require.True(t, run.IsReplay)

	node, err = models.FindCanvasNode(database.Conn(), canvas.ID, componentNodeID)
	require.NoError(t, err)
	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, componentNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 2, "the replay must have produced its own execution")

	var replayExecution *models.CanvasNodeExecution
	for i := range executions {
		if executions[i].RunID == run.ID {
			replayExecution = &executions[i]
		}
	}
	require.NotNil(t, replayExecution)
	require.Equal(t, models.CanvasNodeExecutionResultFailed, replayExecution.Result)
	require.Equal(t, models.CanvasNodeExecutionResultReasonError, replayExecution.ResultReason,
		"fixture sanity: the replay must have failed while building its configuration")

	onErrorEvents, err = models.ListCanvasEvents(database.Conn(), canvas.ID, onErrorNodeID, 10, nil)
	require.NoError(t, err)
	assert.Len(t, onErrorEvents, 1, "a replay's configuration failure must not notify On Error nodes")
}

// Replaying a node with outgoing edges suppresses downstream fan-out (no
// queue item for the downstream node, output events stored and routed), and
// the replay run still reaches a terminal state.
func Test__Replay_NoDownstreamFanOutButRunFinalizes(t *testing.T) {
	componentName := "replay-dod7-" + uuid.New().String()

	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name: componentName,
		ExecuteFunc: func(ctx core.ExecutionContext) error {
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "replay.test", []any{map[string]any{"ok": true}})
		},
	}))

	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	nodeAID := "node-a"
	nodeBID := "node-b"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
			{NodeID: nodeAID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}})},
			{NodeID: nodeBID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: nodeAID, Channel: "default"},
			{SourceID: nodeAID, TargetID: nodeBID, Channel: "default"},
		},
	)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, nodeAID)
	require.NoError(t, err)

	var run *models.CanvasRun
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		run, _, txErr = models.CreateMultiInputNodeReplay(tx, node, nil, []models.ReplayInput{{Payload: map[string]any{"a": 1}}})
		return txErr
	})
	require.NoError(t, err)
	require.True(t, run.IsReplay)

	logger := log.NewEntry(log.New())
	queueWorker := NewNodeQueueWorker(r.Registry, r.GitProvider, "")
	require.NoError(t, queueWorker.LockAndProcessNode(logger, *node, time.Now()))

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, nodeAID, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)

	executor := newTestExecutorForReplay(r)
	require.NoError(t, executor.LockAndProcessNodeExecution(executions[0].ID))

	outputEvents, err := models.ListCanvasEvents(database.Conn(), canvas.ID, nodeAID, 10, nil)
	require.NoError(t, err)
	require.Len(t, outputEvents, 1)

	router := NewEventRouter("")
	require.NoError(t, router.LockAndProcessEvent(logger, outputEvents[0], time.Now()))

	//
	// No queue item for the downstream node, output event stored and routed.
	//
	queueItemsForB, err := models.ListNodeQueueItems(database.Conn(), canvas.ID, nodeBID, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItemsForB, "a replay must not fan out to downstream nodes")

	routedEvent, err := models.FindCanvasEvent(outputEvents[0].ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStateRouted, routedEvent.State)

	//
	// The replay run must still reach a terminal state.
	//
	finalizer := NewRunFinalizer("", r.Registry)
	require.NoError(t, finalizer.finalizeRun(canvas.ID, run.ID, runFinalizerTriggerEventTerminal))

	finishedRun, err := models.FindUnscopedCanvasRun(database.Conn(), run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, finishedRun.State, "a replay run must reach a terminal state, not hang forever")
}

// Replaying a node picks up its live configuration, not the original
// execution's configuration snapshot.
func Test__Replay_UsesLiveConfigurationNotOriginalSnapshot(t *testing.T) {
	componentName := "replay-dod9-" + uuid.New().String()

	var capturedConfig any
	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name: componentName,
		ExecuteFunc: func(ctx core.ExecutionContext) error {
			capturedConfig = ctx.Configuration
			return ctx.ExecutionState.Pass()
		},
	}))

	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
			{
				NodeID:        componentNodeID,
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}}),
				Configuration: datatypes.NewJSONType(map[string]any{"greeting": "original"}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	originalEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, map[string]any{"v": "x"})
	originalExecution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, originalEvent.ID, originalEvent.ID)
	originalExecution.Configuration = datatypes.NewJSONType(map[string]any{"greeting": "original"})
	require.NoError(t, database.Conn().Save(originalExecution).Error)
	_, err := originalExecution.Pass(map[string][]any{})
	require.NoError(t, err, "the original execution is history: it is already finished by the time it is replayed")

	//
	// Edit the node configuration after the original execution.
	//
	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNodeID)
	require.NoError(t, err)
	node.Configuration = datatypes.NewJSONType(map[string]any{"greeting": "edited"})
	require.NoError(t, database.Conn().Save(node).Error)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, txErr := models.CreateMultiInputNodeReplay(tx, node, &originalExecution.ID, []models.ReplayInput{{
			Payload:           map[string]any{"v": "y"},
			SourceExecutionID: &originalExecution.ID,
		}})
		return txErr
	})
	require.NoError(t, err)

	logger := log.NewEntry(log.New())
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, "")
	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, componentNodeID, []string{models.CanvasNodeExecutionStatePending}, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, map[string]any{"greeting": "edited"}, executions[0].Configuration.Data())

	executor := newTestExecutorForReplay(r)
	require.NoError(t, executor.LockAndProcessNodeExecution(executions[0].ID))

	assert.Equal(t, map[string]any{"greeting": "edited"}, capturedConfig)
}

// A replay of a node that is already at its concurrency limit because a
// live execution is in flight is queued on the node and runs only after
// that execution is released - it does not run concurrently with it.
func Test__Replay_QueuedWhileNodeBusyRunsAfterRelease(t *testing.T) {
	componentName := "replay-dod11-" + uuid.New().String()

	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:        componentName,
		ExecuteFunc: func(ctx core.ExecutionContext) error { return ctx.ExecutionState.Pass() },
	}))

	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "component-1"

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
			{NodeID: componentNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}})},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	//
	// Put a live execution in flight first, exactly as normal processing
	// would: consume a queue item, which creates the execution that takes
	// the node's only concurrency slot.
	//
	liveEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNodeID, "default", nil)
	support.CreateQueueItem(t, canvas.ID, componentNodeID, liveEvent.ID, liveEvent.ID)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNodeID)
	require.NoError(t, err)

	logger := log.NewEntry(log.New())
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, "")
	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	inFlight, err := models.ListNodeExecutions(database.Conn(), canvas.ID, componentNodeID, models.CanvasNodeExecutionActiveStates, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, inFlight, 1, "node must be busy with the live execution before the replay is created")

	//
	// Now create the replay while the node is busy.
	//
	var queueItems []*models.CanvasNodeQueueItem
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		_, queueItems, txErr = models.CreateMultiInputNodeReplay(tx, node, nil, []models.ReplayInput{{Payload: map[string]any{"a": 1}}})
		return txErr
	})
	require.NoError(t, err)

	//
	// Attempting to process now must skip: the node is not ready, so the
	// replay must not run concurrently with the live execution.
	//
	err = worker.LockAndProcessNode(logger, *node, time.Now())
	require.NoError(t, err)

	stillQueued, err := models.FindNodeQueueItem(canvas.ID, queueItems[0].ID)
	require.NoError(t, err, "replay queue item must still be pending - it must not have been processed while the node was busy")
	assert.Equal(t, queueItems[0].ID, stillQueued.ID)

	liveExecutions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, componentNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, liveExecutions, 1, "only the live execution should exist while the node is busy - the replay must not have started concurrently")

	//
	// Release the live execution. The node becomes ready again, and only
	// then does the replay get processed.
	//
	executor := newTestExecutorForReplay(r)
	require.NoError(t, executor.LockAndProcessNodeExecution(liveExecutions[0].ID))

	inFlight, err = models.ListNodeExecutions(database.Conn(), canvas.ID, componentNodeID, models.CanvasNodeExecutionActiveStates, nil, 10, nil)
	require.NoError(t, err)
	require.Empty(t, inFlight, "the live execution must have released the node's concurrency slot")

	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	_, err = models.FindNodeQueueItem(canvas.ID, queueItems[0].ID)
	assert.Error(t, err, "replay queue item must have been consumed once the node became ready")

	allExecutions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, componentNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	assert.Len(t, allExecutions, 2, "the replay execution must have been created only after the live one released the node")
}

// A completed replay pins a copy of its payload on the replay
// record, and replaying that replay still succeeds after the original input
// event has been removed by retention.
func Test__Replay_PinnedPayloadSurvivesRetentionOfOriginalEvent(t *testing.T) {
	componentName := "replay-dod12-" + uuid.New().String()

	var capturedData any
	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name: componentName,
		ExecuteFunc: func(ctx core.ExecutionContext) error {
			capturedData = ctx.Data
			return ctx.ExecutionState.Pass()
		},
	}))

	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
			{NodeID: componentNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}})},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	originalEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, map[string]any{"v": "original"})
	originalExecution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, originalEvent.ID, originalEvent.ID)
	_, err := originalExecution.Pass(map[string][]any{})
	require.NoError(t, err, "the original execution is history: it is already finished by the time it is replayed")

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNodeID)
	require.NoError(t, err)

	pinnedPayload := map[string]any{"v": "replay-1-payload"}

	//
	// Create and run replay 1.
	//
	var run1 *models.CanvasRun
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		run1, _, txErr = models.CreateMultiInputNodeReplay(tx, node, &originalExecution.ID, []models.ReplayInput{{
			Payload:           pinnedPayload,
			SourceExecutionID: &originalExecution.ID,
		}})
		return txErr
	})
	require.NoError(t, err)

	logger := log.NewEntry(log.New())
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, "")
	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	executions1, err := models.ListNodeExecutions(database.Conn(), canvas.ID, componentNodeID, []string{models.CanvasNodeExecutionStatePending}, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions1, 1)
	replay1Execution := executions1[0]

	executor := newTestExecutorForReplay(r)
	require.NoError(t, executor.LockAndProcessNodeExecution(replay1Execution.ID))

	//
	// Retention deletes the ORIGINAL input event.
	//
	require.NoError(t, database.Conn().Exec("DELETE FROM workflow_events WHERE id = ?", originalEvent.ID).Error)

	//
	// Replay that replay, using the payload pinned on run1's own record
	// (not the now-deleted original event).
	//
	freshRun1, err := models.FindUnscopedCanvasRun(database.Conn(), run1.ID)
	require.NoError(t, err)
	require.NotNil(t, freshRun1.ReplayPayload.Data())

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, txErr := models.CreateMultiInputNodeReplay(tx, node, &replay1Execution.ID, []models.ReplayInput{{
			Payload:           freshRun1.ReplayPayload.Data(),
			SourceExecutionID: &replay1Execution.ID,
		}})
		return txErr
	})
	require.NoError(t, err, "replaying a replay must succeed even after the original input event was deleted by retention")

	require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))

	executions2, err := models.ListNodeExecutions(database.Conn(), canvas.ID, componentNodeID, []string{models.CanvasNodeExecutionStatePending}, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions2, 1)
	replay2Execution := &executions2[0]

	require.NoError(t, executor.LockAndProcessNodeExecution(replay2Execution.ID))

	assert.Equal(t, pinnedPayload, capturedData, "the second-level replay must use the payload pinned on the first replay's record")
}

// buildOriginalMergeExecution builds a real merge execution the way
// Test__NodeQueueWorker_MergeExecutionRecordsConsumedEventsFromThreeSources
// does: one root event and one event per distinct source, all queued for the
// merge node and processed one at a time through the real worker loop.
func buildOriginalMergeExecution(t *testing.T, canvasID uuid.UUID, worker *NodeQueueWorker, logger *log.Entry, mergeNodeID string, sourceNodeIDs []string) *models.CanvasNodeExecution {
	rootEvent := support.EmitCanvasEventForNode(t, canvasID, "trigger-1", "default", nil)
	for _, sourceID := range sourceNodeIDs {
		event := support.EmitCanvasEventForNode(t, canvasID, sourceID, "default", nil)
		support.CreateQueueItem(t, canvasID, mergeNodeID, rootEvent.ID, event.ID)
	}

	for range sourceNodeIDs {
		node, err := models.FindCanvasNode(database.Conn(), canvasID, mergeNodeID)
		require.NoError(t, err)
		require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))
	}

	executions, err := models.ListNodeExecutions(database.Conn(), canvasID, mergeNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1, "the original merge execution must have accumulated all sources into one execution")
	require.Equal(t, models.CanvasNodeExecutionStateFinished, executions[0].State, "the original execution must be history: already finished")

	return &executions[0]
}

// replayInputsFromHistory is what a caller does in production to turn an
// aggregating node's surviving consumed events into CreateMultiInputNodeReplay's
// inputs - CreateMultiInputNodeReplay itself never reads history. skipSourceNodeID, if non-empty, simulates retention having
// deleted that one source's surviving event.
func replayInputsFromHistory(t *testing.T, executionID uuid.UUID, skipSourceNodeID string) []models.ReplayInput {
	consumed, err := models.ListConsumedEventsForExecution(database.Conn(), executionID)
	require.NoError(t, err)

	inputs := make([]models.ReplayInput, 0, len(consumed))
	for _, c := range consumed {
		require.NotNil(t, c.EventID)
		event, err := models.FindCanvasEvent(*c.EventID)
		require.NoError(t, err)
		if event.NodeID == skipSourceNodeID {
			continue
		}
		nodeID := event.NodeID
		inputs = append(inputs, models.ReplayInput{
			Payload:      event.Data.Data(),
			SourceNodeID: &nodeID,
		})
	}

	return inputs
}

// Given a merge node whose original execution consumed N input
// events, replaying it feeds all N and the new execution reaches Finished
// rather than staying Started.
func Test__Replay_MultiInputReplayFeedsAllSurvivingInputsAndFinishes(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	source1, source2, source3 := "source-1", "source-2", "source-3"
	mergeNodeID := "merge-node"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
			{NodeID: source1, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: source2, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: source3, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: mergeNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}})},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: source1, Channel: "default"},
			{SourceID: triggerNodeID, TargetID: source2, Channel: "default"},
			{SourceID: triggerNodeID, TargetID: source3, Channel: "default"},
			{SourceID: source1, TargetID: mergeNodeID, Channel: "default"},
			{SourceID: source2, TargetID: mergeNodeID, Channel: "default"},
			{SourceID: source3, TargetID: mergeNodeID, Channel: "default"},
		},
	)

	logger := log.NewEntry(log.New())
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, "")

	originalExecution := buildOriginalMergeExecution(t, canvas.ID, worker, logger, mergeNodeID, []string{source1, source2, source3})

	inputs := replayInputsFromHistory(t, originalExecution.ID, "")
	require.Len(t, inputs, 3, "all 3 of the original execution's consumed events must have survived")

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, mergeNodeID)
	require.NoError(t, err)

	var replayRun *models.CanvasRun
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		replayRun, _, txErr = models.CreateMultiInputNodeReplay(tx, node, &originalExecution.ID, inputs)
		return txErr
	})
	require.NoError(t, err, "a replay covering every distinct incoming source must not be refused")

	for range inputs {
		node, err = models.FindCanvasNode(database.Conn(), canvas.ID, mergeNodeID)
		require.NoError(t, err)
		require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))
	}

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, mergeNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 2, "the original execution plus exactly one new replay execution")

	var replayExecution *models.CanvasNodeExecution
	for i := range executions {
		if executions[i].RunID == replayRun.ID {
			replayExecution = &executions[i]
		}
	}
	require.NotNil(t, replayExecution, "a merge execution must have been created under the replay run")

	assert.Equal(t, models.CanvasNodeExecutionStateFinished, replayExecution.State,
		"the replay must feed all N inputs and reach Finished, not stay Started waiting for inputs that already arrived")

	replayConsumed, err := models.ListConsumedEventsForExecution(database.Conn(), replayExecution.ID)
	require.NoError(t, err)
	assert.Len(t, replayConsumed, len(inputs), "the replay execution must have consumed all N replay inputs")
}

// Given a merge node whose original execution has fewer surviving
// input events than distinct incoming sources, the replay is refused with a
// message naming the missing input(s), instead of creating an execution
// that can never finish.
func Test__Replay_RefusesMultiInputReplayMissingASurvivingSource(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	source1, source2, source3 := "source-1", "source-2", "source-3"
	mergeNodeID := "merge-node"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
			{NodeID: source1, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: source2, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: source3, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: mergeNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}})},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: source1, Channel: "default"},
			{SourceID: triggerNodeID, TargetID: source2, Channel: "default"},
			{SourceID: triggerNodeID, TargetID: source3, Channel: "default"},
			{SourceID: source1, TargetID: mergeNodeID, Channel: "default"},
			{SourceID: source2, TargetID: mergeNodeID, Channel: "default"},
			{SourceID: source3, TargetID: mergeNodeID, Channel: "default"},
		},
	)

	logger := log.NewEntry(log.New())
	worker := NewNodeQueueWorker(r.Registry, r.GitProvider, "")

	originalExecution := buildOriginalMergeExecution(t, canvas.ID, worker, logger, mergeNodeID, []string{source1, source2, source3})

	//
	// Simulate retention having deleted source-3's surviving input event:
	// only 2 of the node's 3 distinct incoming sources have a usable input.
	//
	inputs := replayInputsFromHistory(t, originalExecution.ID, source3)
	require.Len(t, inputs, 2)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, mergeNodeID)
	require.NoError(t, err)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, txErr := models.CreateMultiInputNodeReplay(tx, node, &originalExecution.ID, inputs)
		return txErr
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrReplayMissingInputs), "must be the ErrReplayMissingInputs sentinel so callers can map it with errors.Is")
	assert.Contains(t, err.Error(), source3, "the refusal message must name the missing source node")

	//
	// Nothing must have been created: only the original execution exists,
	// and no replay run/queue item was left behind.
	//
	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, mergeNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	assert.Len(t, executions, 1, "a refused replay must not create an execution that can never finish")

	queueItems, err := models.ListNodeQueueItems(database.Conn(), canvas.ID, mergeNodeID, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)
}
