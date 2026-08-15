package canvases

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func Test__ReplayNode_ValidRequest_ReturnsRunAndQueueItemIDs(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	sourceNodeID := "source-1"
	targetNodeID := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: sourceNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: targetNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: sourceNodeID, TargetID: targetNodeID, Channel: "default"},
		},
	)

	response, err := ReplayNode(
		context.Background(),
		database.DB(t.Context()),
		canvas,
		targetNodeID,
		nil,
		map[string]any{"v": "replay-payload"},
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, response)

	require.NotEmpty(t, response.RunId)
	runID, err := uuid.Parse(response.RunId)
	require.NoError(t, err)

	require.Len(t, response.QueueItemIds, 1)
	queueItemID, err := uuid.Parse(response.QueueItemIds[0])
	require.NoError(t, err)

	run, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, runID)
	require.NoError(t, err)
	assert.True(t, run.IsReplay)

	var queueItem models.CanvasNodeQueueItem
	require.NoError(t, database.Conn().Where("id = ?", queueItemID).First(&queueItem).Error)
	assert.Equal(t, targetNodeID, queueItem.NodeID)
	assert.Equal(t, runID, queueItem.RunID)

	// No execution was created synchronously - that is NodeQueueWorker's job.
	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, targetNodeID, nil, nil, 100, nil)
	require.NoError(t, err)
	assert.Empty(t, executions)
}

func Test__ReplayNode_UnknownNodeID_404AndNothingCreated(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	runsBefore, err := models.ListCanvasRuns(canvas.ID, 100, nil, models.CanvasRunFilters{})
	require.NoError(t, err)
	require.Empty(t, runsBefore)

	response, err := ReplayNode(
		context.Background(),
		database.DB(t.Context()),
		canvas,
		"does-not-exist",
		nil,
		map[string]any{"v": 1},
		nil,
		nil,
	)

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	assert.Contains(t, err.Error(), "not found")

	runsAfter, err := models.ListCanvasRuns(canvas.ID, 100, nil, models.CanvasRunFilters{})
	require.NoError(t, err)
	assert.Empty(t, runsAfter, "no run should have been created for an unknown node_id")
}

func Test__ReplayNode_TriggerNode_400AndNothingCreated(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
		},
		[]models.Edge{},
	)

	response, err := ReplayNode(
		context.Background(),
		database.DB(t.Context()),
		canvas,
		triggerNodeID,
		nil,
		map[string]any{"v": 1},
		nil,
		nil,
	)

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.ErrorIs(t, err, models.ErrReplayTriggerNode)

	runs, err := models.ListCanvasRuns(canvas.ID, 100, nil, models.CanvasRunFilters{})
	require.NoError(t, err)
	assert.Empty(t, runs)
}

func Test__ReplayNode_SourceExecutionWithMultipleSources_DistinctInputsPerSource(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	source1, source2 := "source-1", "source-2"
	mergeNodeID := "merge-node"
	targetNodeID := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: source1, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: source2, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: mergeNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}})},
			{NodeID: targetNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: source1, TargetID: mergeNodeID, Channel: "default"},
			{SourceID: source2, TargetID: mergeNodeID, Channel: "default"},
			{SourceID: mergeNodeID, TargetID: targetNodeID, Channel: "default"},
		},
	)

	event1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, source1, "default", nil, map[string]any{"v": "from-source-1"})
	event2 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, source2, "default", nil, map[string]any{"v": "from-source-2"})

	// This is the merge node's own (non-replay) execution, standing in for
	// what a real merge run would have produced - it consumed one event from
	// each of its two distinct sources.
	mergeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, mergeNodeID, event1.ID, event1.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, event1.ID))
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, event2.ID))

	// Replay target is deliberately NOT the aggregating merge node itself:
	// target-1 has no aggregating-node source-coverage check, so a collapse
	// bug here can only be caught by this test's own distinctness
	// assertion below, not by models.CreateMultiInputNodeReplay's guard.
	response, err := ReplayNode(
		context.Background(),
		database.DB(t.Context()),
		canvas,
		targetNodeID,
		&mergeExecution.ID,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.QueueItemIds, 2, "one queue item per surviving consumed event")

	gotSourceNodes := make([]string, 0, 2)
	gotPayloadBySource := make(map[string]any, 2)
	for _, queueItemID := range response.QueueItemIds {
		id, err := uuid.Parse(queueItemID)
		require.NoError(t, err)

		var queueItem models.CanvasNodeQueueItem
		require.NoError(t, database.Conn().Where("id = ?", id).First(&queueItem).Error)

		event, err := models.FindCanvasEvent(queueItem.EventID)
		require.NoError(t, err)

		gotSourceNodes = append(gotSourceNodes, event.NodeID)
		gotPayloadBySource[event.NodeID] = event.Data.Data()
	}

	assert.ElementsMatch(t, []string{source1, source2}, gotSourceNodes,
		"each input must keep its own distinct source node, not collapse onto one (replay_node.go:156-162)")
	assert.Equal(t, map[string]any{"v": "from-source-1"}, gotPayloadBySource[source1])
	assert.Equal(t, map[string]any{"v": "from-source-2"}, gotPayloadBySource[source2])
}

func Test__ReplayNode_SourceExecutionIsReplay_UsesRunPinnedPayload(t *testing.T) {
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
			{NodeID: componentNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	originalEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, map[string]any{"v": "original"})
	originalExecution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, originalEvent.ID, originalEvent.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), originalExecution.ID, originalEvent.ID))

	// First-level replay: resolved from originalExecution's history.
	response1, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, componentNodeID, &originalExecution.ID, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, response1.QueueItemIds, 1)

	run1ID, err := uuid.Parse(response1.RunId)
	require.NoError(t, err)
	run1, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run1ID)
	require.NoError(t, err)
	require.True(t, run1.IsReplay)

	queueItem1ID, err := uuid.Parse(response1.QueueItemIds[0])
	require.NoError(t, err)
	var queueItem1 models.CanvasNodeQueueItem
	require.NoError(t, database.Conn().Where("id = ?", queueItem1ID).First(&queueItem1).Error)

	// Materialize run1's execution the way NodeQueueWorker would - this test
	// stays at the gRPC-action layer, so it builds the row directly instead
	// of running the worker loop. BeforeCreate resolves RunID from
	// RootEventID, which correctly lands on run1 because
	// CreateMultiInputNodeReplay set the input event's RunID explicitly to
	// run1.ID (see canvas_node_replay.go's comment on why).
	replay1Execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, queueItem1.RootEventID, queueItem1.EventID)
	require.Equal(t, run1.ID, replay1Execution.RunID, "fixture sanity: replay1Execution must belong to run1")

	// Delete the ORIGINAL, pre-replay input event - the strongest form of
	// proof that a replay-of-a-replay does not need to re-read that far
	// back into history. replay1's own input event (which
	// ResolveReplaySourceNodeID separately reads for source-node
	// attribution) is left intact.
	require.NoError(t, database.Conn().Exec("DELETE FROM workflow_events WHERE id = ?", originalEvent.ID).Error)

	// Second-level replay: replay1Execution has zero consumed events of its
	// own (no worker ever processed it) and the original event is gone, so
	// this can only succeed via run1.ReplayPayload.
	response2, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, componentNodeID, &replay1Execution.ID, nil, nil, nil)
	require.NoError(t, err, "replay-of-a-replay must succeed via the run's pinned ReplayPayload, not by re-reading the (now-deleted) input event")
	require.Len(t, response2.QueueItemIds, 1)

	queueItem2ID, err := uuid.Parse(response2.QueueItemIds[0])
	require.NoError(t, err)
	var queueItem2 models.CanvasNodeQueueItem
	require.NoError(t, database.Conn().Where("id = ?", queueItem2ID).First(&queueItem2).Error)

	event2, err := models.FindCanvasEvent(queueItem2.EventID)
	require.NoError(t, err)

	assert.Equal(t, run1.ReplayPayload.Data(), event2.Data.Data(),
		"the second-level replay's input must be run1's pinned ReplayPayload, not a re-read of history")
	assert.Equal(t, map[string]any{"v": "original"}, event2.Data.Data())
}

func Test__ReplayNode_SourceExecutionNoConsumedEvents_400AndNothingCreated(t *testing.T) {
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
			{NodeID: componentNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNodeID, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID)
	// Deliberately no models.CreateConsumedEvent call: this execution has no
	// surviving consumed events at all.

	runsBefore, err := models.ListCanvasRuns(canvas.ID, 100, nil, models.CanvasRunFilters{})
	require.NoError(t, err)

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, componentNodeID, &execution.ID, nil, nil, nil)

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.Contains(t, err.Error(), "no surviving consumed events")

	runsAfter, err := models.ListCanvasRuns(canvas.ID, 100, nil, models.CanvasRunFilters{})
	require.NoError(t, err)

	runIDsBefore := make([]uuid.UUID, len(runsBefore))
	for i, run := range runsBefore {
		runIDsBefore[i] = run.ID
	}
	runIDsAfter := make([]uuid.UUID, len(runsAfter))
	for i, run := range runsAfter {
		runIDsAfter[i] = run.ID
	}
	assert.ElementsMatch(t, runIDsBefore, runIDsAfter, "no run should have been created for a source execution with no surviving consumed events")
}

func createJoinCanvas(t *testing.T, r *support.ResourceRegistry) (*models.Canvas, string, string, string) {
	laneA, laneB := "lane-a", "lane-b"
	joinNodeID := "join"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: laneA, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: laneB, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: joinNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}})},
		},
		[]models.Edge{
			{SourceID: laneA, TargetID: joinNodeID, Channel: "default"},
			{SourceID: laneB, TargetID: joinNodeID, Channel: "default"},
		},
	)
	return canvas, laneA, laneB, joinNodeID
}

func Test__ReplayNode_MissingOneOfTwoInputs_400NamesMissingSource(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	explicitInputs := []models.ReplayInput{
		{Payload: map[string]any{"v": "from-a"}, SourceNodeID: &laneA},
	}

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, nil, nil, nil, explicitInputs)
	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.ErrorIs(t, err, models.ErrReplayMissingInputs)
	assert.Contains(t, err.Error(), laneB, "the refusal must name the missing lane-b")
}

func Test__ReplayNode_InputsWinButSourceExecutionIDStillSetsLineage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	sourceNodeID := "source-1"
	targetNodeID := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: sourceNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: targetNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: sourceNodeID, TargetID: targetNodeID, Channel: "default"},
		},
	)

	// A real execution row is required (replay_source_execution_id has a
	// foreign key), but it carries no history relevant to this replay's
	// payload: proves inputs wins over history resolution, since resolving
	// from this execution's own (empty) consumed events would either fail or
	// produce a different payload than "from-inputs".
	unrelatedEvent := support.EmitCanvasEventForNode(t, canvas.ID, sourceNodeID, "default", nil)
	lineageExecution := support.CreateCanvasNodeExecution(t, canvas.ID, targetNodeID, unrelatedEvent.ID, unrelatedEvent.ID)
	lineageSourceExecutionID := lineageExecution.ID
	explicitInputs := []models.ReplayInput{
		{Payload: map[string]any{"v": "from-inputs"}, SourceNodeID: &sourceNodeID},
	}

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, targetNodeID, &lineageSourceExecutionID, nil, nil, explicitInputs)
	require.NoError(t, err)
	require.Len(t, response.QueueItemIds, 1)

	runID, err := uuid.Parse(response.RunId)
	require.NoError(t, err)
	run, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, runID)
	require.NoError(t, err)
	require.NotNil(t, run.ReplaySourceExecutionID)
	assert.Equal(t, lineageSourceExecutionID, *run.ReplaySourceExecutionID)

	queueItemID, err := uuid.Parse(response.QueueItemIds[0])
	require.NoError(t, err)
	var queueItem models.CanvasNodeQueueItem
	require.NoError(t, database.Conn().Where("id = ?", queueItemID).First(&queueItem).Error)
	event, err := models.FindCanvasEvent(queueItem.EventID)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"v": "from-inputs"}, event.Data.Data(), "the input must come from `inputs`, not from history resolution")
}

// requireNoReplayRunExists guards the "nothing was created" half of a refusal.
// Runs are asserted by kind rather than by count because emitting a fixture
// event creates an ordinary run of its own.
func requireNoReplayRunExists(t *testing.T, canvasID uuid.UUID) {
	runs, err := models.ListCanvasRuns(canvasID, 100, nil, models.CanvasRunFilters{})
	require.NoError(t, err)
	for _, run := range runs {
		assert.False(t, run.IsReplay, "a refused replay must not have created a replay run")
	}
}

func Test__ReplayNode_RetentionErasedOneInput_400NamesTheUncoveredSource(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	eventA := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneA, "default", nil, map[string]any{"v": "from-a"})
	eventB := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneB, "default", nil, map[string]any{"v": "from-b"})
	mergeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, joinNodeID, eventA.ID, eventA.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventA.ID))
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventB.ID))

	require.NoError(t, database.Conn().Where("id = ?", eventB.ID).Delete(&models.CanvasEvent{}).Error)

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, &mergeExecution.ID, nil, nil, nil)
	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.ErrorIs(t, err, models.ErrReplayMissingInputs)
	assert.Contains(t, err.Error(), laneB, "the refusal must name the source whose input retention erased")

	requireNoReplayRunExists(t, canvas.ID)
}

func Test__ReplayNode_RetentionErasedEveryInput_400AndNothingCreated(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	eventA := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneA, "default", nil, map[string]any{"v": "from-a"})
	eventB := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneB, "default", nil, map[string]any{"v": "from-b"})
	mergeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, joinNodeID, eventA.ID, eventA.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventA.ID))
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventB.ID))

	require.NoError(t, database.Conn().Where("id IN ?", []uuid.UUID{eventA.ID, eventB.ID}).Delete(&models.CanvasEvent{}).Error)

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, &mergeExecution.ID, nil, nil, nil)
	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.Contains(t, err.Error(), "no surviving consumed events")

	requireNoReplayRunExists(t, canvas.ID)
}

func Test__ReplayNode_SourceThatDoesNotFeedTheNode_400AndNothingCreated(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	// lane-b stops feeding the join node, so only a non-browser caller can
	// still name it: the modal strips detached inputs before launching.
	setLiveVersionEdges(t, canvas, []models.Edge{{SourceID: laneA, TargetID: joinNodeID, Channel: "default"}})

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, nil, nil, nil, []models.ReplayInput{
		{Payload: map[string]any{"v": "from-a"}, SourceNodeID: &laneA},
		{Payload: map[string]any{"v": "from-b"}, SourceNodeID: &laneB},
	})
	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.Contains(t, err.Error(), laneB)
	assert.Contains(t, err.Error(), "is not an incoming source")

	legacyResponse, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, nil, map[string]any{"v": "from-b"}, &laneB, nil)
	require.Error(t, err, "the legacy single-payload path must refuse the same source")
	assert.Nil(t, legacyResponse)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.Contains(t, err.Error(), laneB)

	requireNoReplayRunExists(t, canvas.ID)
}

func Test__ReplayNode_SourceExecutionFromAnotherCanvas_404AndNothingCreated(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	otherCanvas, otherLaneA, _, otherJoinNodeID := createJoinCanvas(t, r)
	otherEvent := support.EmitCanvasEventForNodeWithData(t, otherCanvas.ID, otherLaneA, "default", nil, map[string]any{"v": "other-canvas"})
	otherExecution := support.CreateCanvasNodeExecution(t, otherCanvas.ID, otherJoinNodeID, otherEvent.ID, otherEvent.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), otherExecution.ID, otherEvent.ID))

	canvas, laneA, _, joinNodeID := createJoinCanvas(t, r)

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, &otherExecution.ID, nil, nil, nil)
	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))

	// The same id is refused even when the caller supplies its own inputs, so
	// lineage cannot smuggle in a reference to another canvas' execution.
	withInputs, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, &otherExecution.ID, nil, nil, []models.ReplayInput{
		{Payload: map[string]any{"v": "from-a"}, SourceNodeID: &laneA},
	})
	require.Error(t, err)
	assert.Nil(t, withInputs)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))

	requireNoReplayRunExists(t, canvas.ID)
}

func Test__ReplayNode_OversizedPayload_400AndNothingCreated(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	oversized := map[string]any{"v": strings.Repeat("x", MaxReplayPayloadSize)}

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, nil, nil, nil, []models.ReplayInput{
		{Payload: map[string]any{"v": "from-a"}, SourceNodeID: &laneA},
		{Payload: oversized, SourceNodeID: &laneB},
	})
	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.Contains(t, err.Error(), "exceeds the maximum")

	events, err := models.ListCanvasEvents(database.Conn(), canvas.ID, laneA, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, events, "a refused replay must not have stored its other inputs' events")

	requireNoReplayRunExists(t, canvas.ID)
}

func Test__ReplayNode_TooManyInputs_400AndNothingCreated(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	inputs := make([]models.ReplayInput, 0, MaxReplayInputs+1)
	inputs = append(inputs, models.ReplayInput{Payload: map[string]any{"v": "from-b"}, SourceNodeID: &laneB})
	for i := 0; i < MaxReplayInputs; i++ {
		inputs = append(inputs, models.ReplayInput{Payload: map[string]any{"v": i}, SourceNodeID: &laneA})
	}

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, nil, nil, nil, inputs)
	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.Contains(t, err.Error(), "exceeds the maximum")

	requireNoReplayRunExists(t, canvas.ID)

	queueItems, err := models.ListNodeQueueItems(database.Conn(), canvas.ID, joinNodeID, 100, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)
}

func Test__ReplayNode_UnattributedBareList_400NamesEverySource(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvas.ID)
	require.NoError(t, err)

	now := time.Now()
	// Simulates a run written before per-input attribution: ReplayPayload is a
	// bare list of payloads, with no record of which source each came from.
	oldFormatRun := &models.CanvasRun{
		WorkflowID:    canvas.ID,
		NodeID:        joinNodeID,
		VersionID:     liveVersion.ID,
		State:         models.CanvasRunStateFinished,
		IsReplay:      true,
		ReplayPayload: models.NewJSONValue([]any{map[string]any{"v": "from-a"}, map[string]any{"v": "from-b"}}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	require.NoError(t, database.Conn().Create(oldFormatRun).Error)

	// root_event_id/event_id are foreign keys, so they must point at real
	// event rows - their content is irrelevant here, since RunID is set
	// explicitly and ReplayInputsFromRun reads the run's own ReplayPayload,
	// not this event.
	backingEvent := support.EmitCanvasEventForNode(t, canvas.ID, laneA, "default", nil)
	oldFormatExecution := &models.CanvasNodeExecution{
		WorkflowID:    canvas.ID,
		NodeID:        joinNodeID,
		RootEventID:   backingEvent.ID,
		EventID:       backingEvent.ID,
		RunID:         oldFormatRun.ID,
		State:         models.CanvasNodeExecutionStateFinished,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	require.NoError(t, database.Conn().Create(oldFormatExecution).Error)

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, &oldFormatExecution.ID, nil, nil, nil)
	require.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.ErrorIs(t, err, models.ErrReplayMissingInputs)
	assert.Contains(t, err.Error(), laneA)
	assert.Contains(t, err.Error(), laneB)
}
