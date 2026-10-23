package canvases

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

// setLiveVersionEdges overwrites the live canvas version's edge list, so a
// test can simulate a node gaining or losing an incoming source after a
// source execution's history was written.
func setLiveVersionEdges(t *testing.T, canvas *models.Canvas, edges []models.Edge) {
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvas.ID)
	require.NoError(t, err)
	liveVersion.Edges = datatypes.NewJSONSlice(edges)
	require.NoError(t, database.Conn().Save(liveVersion).Error)
}

func resolvedInputBySource(t *testing.T, inputs []*pb.ResolvedReplayInput, sourceNodeID string) *pb.ResolvedReplayInput {
	for _, input := range inputs {
		if input.SourceNodeId == sourceNodeID {
			return input
		}
	}
	require.Fail(t, "no resolved input for source", sourceNodeID)
	return nil
}

func Test__ResolveReplayInputs_MergeTwoSurvivingConsumedEvents_TwoRecoveredInputs(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	eventA := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneA, "default", nil, map[string]any{"v": "from-a"})
	eventB := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneB, "default", nil, map[string]any{"v": "from-b"})
	mergeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, joinNodeID, eventA.ID, eventA.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventA.ID))
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventB.ID))

	response, err := ResolveReplayInputs(context.Background(), database.DB(t.Context()), canvas, joinNodeID, mergeExecution.ID)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Inputs, 2, "one input per distinct incoming source")

	a := resolvedInputBySource(t, response.Inputs, laneA)
	assert.Equal(t, pb.ResolvedReplayInput_STATUS_RECOVERED, a.Status)
	require.NotNil(t, a.Payload)
	assert.Equal(t, "from-a", a.Payload.AsMap()["v"])

	b := resolvedInputBySource(t, response.Inputs, laneB)
	assert.Equal(t, pb.ResolvedReplayInput_STATUS_RECOVERED, b.Status)
	require.NotNil(t, b.Payload)
	assert.Equal(t, "from-b", b.Payload.AsMap()["v"], "each input must carry its own distinct payload, not the other lane's")
}

func Test__ResolveReplayInputs_ThirdIncomingSourceIsMissingWithEmptyPayload(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	eventA := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneA, "default", nil, map[string]any{"v": "from-a"})
	eventB := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneB, "default", nil, map[string]any{"v": "from-b"})
	mergeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, joinNodeID, eventA.ID, eventA.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventA.ID))
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventB.ID))

	laneC := "lane-c"
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvas.ID)
	require.NoError(t, err)
	setLiveVersionEdges(t, canvas, append(slices.Clone([]models.Edge(liveVersion.Edges)), models.Edge{SourceID: laneC, TargetID: joinNodeID, Channel: "default"}))

	response, err := ResolveReplayInputs(context.Background(), database.DB(t.Context()), canvas, joinNodeID, mergeExecution.ID)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Inputs, 3, "must include the new third source, not silently omit it")

	c := resolvedInputBySource(t, response.Inputs, laneC)
	assert.Equal(t, pb.ResolvedReplayInput_STATUS_MISSING, c.Status)
	assert.Nil(t, c.Payload, "a missing input's payload must be empty")

	// the two originally-recovered lanes are unaffected
	assert.Equal(t, pb.ResolvedReplayInput_STATUS_RECOVERED, resolvedInputBySource(t, response.Inputs, laneA).Status)
	assert.Equal(t, pb.ResolvedReplayInput_STATUS_RECOVERED, resolvedInputBySource(t, response.Inputs, laneB).Status)
}

func Test__ResolveReplayInputs_SourceNoLongerIncoming_MarkedDetached(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	eventA := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneA, "default", nil, map[string]any{"v": "from-a"})
	eventB := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneB, "default", nil, map[string]any{"v": "from-b"})
	mergeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, joinNodeID, eventA.ID, eventA.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventA.ID))
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventB.ID))

	// lane-b's edge into the join node is removed - only lane-a still feeds it.
	setLiveVersionEdges(t, canvas, []models.Edge{{SourceID: laneA, TargetID: joinNodeID, Channel: "default"}})

	response, err := ResolveReplayInputs(context.Background(), database.DB(t.Context()), canvas, joinNodeID, mergeExecution.ID)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Inputs, 2, "lane-a recovered plus lane-b detached, no third slot")

	a := resolvedInputBySource(t, response.Inputs, laneA)
	assert.Equal(t, pb.ResolvedReplayInput_STATUS_RECOVERED, a.Status)

	b := resolvedInputBySource(t, response.Inputs, laneB)
	assert.Equal(t, pb.ResolvedReplayInput_STATUS_DETACHED, b.Status, "lane-b no longer feeds the node, so it must be detached, not recovered")
	require.NotNil(t, b.Payload, "a detached input still carries its recovered payload")
	assert.Equal(t, "from-b", b.Payload.AsMap()["v"])
}

// resolvedInputsAsPairs renders each resolved input as "<source>=<v>", so two
// inputs from the same source stay distinguishable in an order-independent
// comparison.
func resolvedInputsAsPairs(t *testing.T, inputs []*pb.ResolvedReplayInput) []string {
	pairs := make([]string, 0, len(inputs))
	for _, input := range inputs {
		require.NotNil(t, input.Payload, "input for source %s carries no payload", input.SourceNodeId)
		pairs = append(pairs, fmt.Sprintf("%s=%v", input.SourceNodeId, input.Payload.AsMap()["v"]))
	}

	return pairs
}

func Test__ResolveReplayInputs_TwoInputsFromOneSource_BothRecoveredAndMatchWhatALaunchCreates(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	eventA1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneA, "default", nil, map[string]any{"v": "from-a-1"})
	eventA2 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneA, "default", nil, map[string]any{"v": "from-a-2"})
	eventB := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneB, "default", nil, map[string]any{"v": "from-b"})
	mergeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, joinNodeID, eventA1.ID, eventA1.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventA1.ID))
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventA2.ID))
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventB.ID))

	response, err := ResolveReplayInputs(context.Background(), database.DB(t.Context()), canvas, joinNodeID, mergeExecution.ID)
	require.NoError(t, err)
	require.Len(t, response.Inputs, 3, "both lane-a inputs must be kept, not collapsed onto one slot")

	for _, input := range response.Inputs {
		assert.Equal(t, pb.ResolvedReplayInput_STATUS_RECOVERED, input.Status)
	}

	shown := resolvedInputsAsPairs(t, response.Inputs)
	assert.ElementsMatch(t, []string{laneA + "=from-a-1", laneA + "=from-a-2", laneB + "=from-b"}, shown)

	launched, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, &mergeExecution.ID, nil, nil, nil)
	require.NoError(t, err)

	launchedPairs := make([]string, 0, len(launched.QueueItemIds))
	for _, queueItemID := range launched.QueueItemIds {
		id, err := uuid.Parse(queueItemID)
		require.NoError(t, err)

		queueItem, err := models.FindNodeQueueItem(canvas.ID, id)
		require.NoError(t, err)

		event, err := models.FindCanvasEvent(queueItem.EventID)
		require.NoError(t, err)

		payload, ok := event.Data.Data().(map[string]any)
		require.True(t, ok)
		launchedPairs = append(launchedPairs, fmt.Sprintf("%s=%v", event.NodeID, payload["v"]))
	}

	assert.ElementsMatch(t, shown, launchedPairs, "what the modal shows must be exactly what a launch replays")
}

func Test__ResolveReplayInputs_RetentionErasedInput_ReportedExpiredRatherThanMissing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	eventA := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneA, "default", nil, map[string]any{"v": "from-a"})
	eventB := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneB, "default", nil, map[string]any{"v": "from-b"})
	mergeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, joinNodeID, eventA.ID, eventA.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventA.ID))
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventB.ID))

	laneC := "lane-c"
	setLiveVersionEdges(t, canvas, []models.Edge{
		{SourceID: laneA, TargetID: joinNodeID, Channel: "default"},
		{SourceID: laneB, TargetID: joinNodeID, Channel: "default"},
		{SourceID: laneC, TargetID: joinNodeID, Channel: "default"},
	})

	response, err := ResolveReplayInputs(context.Background(), database.DB(t.Context()), canvas, joinNodeID, mergeExecution.ID)
	require.NoError(t, err)
	assert.Equal(t, pb.ResolvedReplayInput_STATUS_MISSING, resolvedInputBySource(t, response.Inputs, laneC).Status,
		"a source this execution never had an input from is missing, not expired")

	require.NoError(t, database.Conn().Where("id = ?", eventB.ID).Delete(&models.CanvasEvent{}).Error)

	consumed, err := models.ListConsumedEventsForExecution(database.Conn(), mergeExecution.ID)
	require.NoError(t, err)
	require.Len(t, consumed, 2, "fixture sanity: ON DELETE SET NULL must leave the link row behind as a tombstone")

	response, err = ResolveReplayInputs(context.Background(), database.DB(t.Context()), canvas, joinNodeID, mergeExecution.ID)
	require.NoError(t, err)
	require.Len(t, response.Inputs, 3)

	assert.Equal(t, pb.ResolvedReplayInput_STATUS_RECOVERED, resolvedInputBySource(t, response.Inputs, laneA).Status)

	erased := resolvedInputBySource(t, response.Inputs, laneB)
	assert.Equal(t, pb.ResolvedReplayInput_STATUS_EXPIRED, erased.Status,
		"an input the execution did consume and retention erased must be expired, not merely missing")
	assert.Nil(t, erased.Payload)
}

func Test__ResolveReplayInputs_EveryInputErasedByRetention_Refused(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	eventA := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneA, "default", nil, map[string]any{"v": "from-a"})
	eventB := support.EmitCanvasEventForNodeWithData(t, canvas.ID, laneB, "default", nil, map[string]any{"v": "from-b"})
	mergeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, joinNodeID, eventA.ID, eventA.ID)
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventA.ID))
	require.NoError(t, models.CreateConsumedEvent(database.Conn(), mergeExecution.ID, eventB.ID))

	require.NoError(t, database.Conn().Where("id IN ?", []uuid.UUID{eventA.ID, eventB.ID}).Delete(&models.CanvasEvent{}).Error)

	response, err := ResolveReplayInputs(context.Background(), database.DB(t.Context()), canvas, joinNodeID, mergeExecution.ID)
	require.Error(t, err, "total loss must be refused outright rather than answered with empty slots")
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.Contains(t, err.Error(), "no surviving consumed events")
}

func Test__ResolveReplayInputs_UnknownSourceExecutionID_RefusedAsNotFound(t *testing.T) {
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

	response, err := ResolveReplayInputs(context.Background(), database.DB(t.Context()), canvas, targetNodeID, uuid.New())
	require.Error(t, err)
	assert.Nil(t, response, "an unknown source execution must be refused outright, never a successful empty list")
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	assert.Contains(t, err.Error(), "not found")
}
