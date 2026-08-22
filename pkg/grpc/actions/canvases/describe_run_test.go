package canvases

import (
	"context"
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
)

func Test__DescribeRun(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns run with root event and execution refs", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{NodeID: "trigger", Type: models.NodeTypeTrigger},
				{NodeID: "node-1", Type: models.NodeTypeComponent},
			},
			[]models.Edge{},
		)

		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
		run := createFinishedRun(t, rootEvent, models.CanvasRunResultPassed)
		execution := createRunExecution(t, run, rootEvent.ID, "node-1", models.CanvasNodeExecutionResultPassed)

		response, err := DescribeRun(context.Background(), database.DB(t.Context()), canvas, run.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Run)

		serializedRun := response.Run
		assert.Equal(t, run.ID.String(), serializedRun.Id)
		assert.Equal(t, run.VersionID.String(), serializedRun.VersionId)
		assert.Equal(t, pb.CanvasRun_STATE_FINISHED, serializedRun.State)
		assert.Equal(t, pb.CanvasRun_RESULT_PASSED, serializedRun.Result)
		require.NotNil(t, serializedRun.RootEvent)
		assert.Equal(t, rootEvent.ID.String(), serializedRun.RootEvent.Id)
		require.Len(t, serializedRun.Executions, 1)
		assert.Equal(t, execution.ID.String(), serializedRun.Executions[0].Id)
	})

	t.Run("returns run with queue items", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{NodeID: "trigger", Type: models.NodeTypeTrigger},
				{NodeID: "node-1", Type: models.NodeTypeComponent},
			},
			[]models.Edge{},
		)

		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "trigger", "default", nil, map[string]any{
			"approval": "waiting",
		})
		run := createStartedRun(t, rootEvent)
		queueItem := support.CreateQueueItem(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID)

		response, err := DescribeRun(context.Background(), database.DB(t.Context()), canvas, run.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Run)
		require.Len(t, response.Run.QueueItems, 1)

		serializedQueueItem := response.Run.QueueItems[0]
		assert.Equal(t, queueItem.ID.String(), serializedQueueItem.Id)
		assert.Equal(t, "node-1", serializedQueueItem.NodeId)
		assert.NotNil(t, serializedQueueItem.CreatedAt)
		require.NotNil(t, serializedQueueItem.RootEvent)
		assert.Equal(t, rootEvent.ID.String(), serializedQueueItem.RootEvent.Id)
		require.NotNil(t, serializedQueueItem.Input)
		assert.Equal(t, "waiting", serializedQueueItem.Input.AsMap()["approval"])
	})

	t.Run("returns pending sub-run without root event", func(t *testing.T) {
		parentCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{NodeID: "trigger", Type: models.NodeTypeTrigger},
				{NodeID: "runApp", Type: models.NodeTypeComponent},
			},
			[]models.Edge{},
		)
		childCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{{NodeID: "onRun", Type: models.NodeTypeTrigger}},
			[]models.Edge{},
		)

		parentRootEvent := support.EmitCanvasEventForNode(t, parentCanvas.ID, "trigger", "default", nil)
		parentRun := createStartedRun(t, parentRootEvent)
		parentExecution := createRunExecution(t, parentRun, parentRootEvent.ID, "runApp", models.CanvasNodeExecutionResultPassed)

		childRun := createSubRunRecord(
			t,
			childCanvas.ID,
			"onRun",
			&parentRun.ID,
			&parentCanvas.ID,
			&parentExecution.ID,
			models.CanvasRunStatePending,
			"",
		)

		response, err := DescribeRun(context.Background(), database.DB(t.Context()), childCanvas, childRun.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Run)
		assert.Equal(t, childRun.ID.String(), response.Run.Id)
		assert.Equal(t, pb.CanvasRun_STATE_PENDING, response.Run.State)
		assert.Nil(t, response.Run.RootEvent)
		require.NotNil(t, response.Run.Parent)
		assert.Equal(t, parentRun.ID.String(), response.Run.Parent.Id)
	})

	t.Run("scopes run to canvas", func(t *testing.T) {
		canvasOne, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})
		canvasTwo, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

		rootEventTwo := support.EmitCanvasEventForNode(t, canvasTwo.ID, "trigger", "default", nil)
		runTwo := createFinishedRun(t, rootEventTwo, models.CanvasRunResultPassed)

		_, err := DescribeRun(context.Background(), database.DB(t.Context()), canvasOne, runTwo.ID.String())
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	})

	t.Run("invalid run id -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

		_, err := DescribeRun(context.Background(), database.DB(t.Context()), canvas, "not-a-uuid")
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("missing run -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

		_, err := DescribeRun(context.Background(), database.DB(t.Context()), canvas, uuid.New().String())
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	})
}

// A multi-input replay writes one input event per source, all belonging to the
// run with no execution of their own, but its queue items and executions are
// rooted on exactly one of them. Serializing any other one leaves the run
// pointing at an event nothing is rooted on, and ListEventExecutions - which
// matches on root_event_id - then answers empty for the whole run.
func Test__DescribeRun__MultiInputReplay__RootEventIsTheOneItsWorkIsRootedOn(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	response, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, nil, nil, nil, []models.ReplayInput{
		{Payload: map[string]any{"v": "from-a"}, SourceNodeID: &laneA},
		{Payload: map[string]any{"v": "from-b"}, SourceNodeID: &laneB},
	})
	require.NoError(t, err)
	require.Len(t, response.QueueItemIds, 2)

	queueItems := make([]*models.CanvasNodeQueueItem, 0, len(response.QueueItemIds))
	for _, id := range response.QueueItemIds {
		queueItemID, err := uuid.Parse(id)
		require.NoError(t, err)

		queueItem, err := models.FindNodeQueueItem(canvas.ID, queueItemID)
		require.NoError(t, err)
		queueItems = append(queueItems, queueItem)
	}

	rootEventID := queueItems[0].RootEventID
	require.Equal(t, rootEventID, queueItems[1].RootEventID, "fixture sanity: both queue items must share one root")

	var otherInputEventID uuid.UUID
	for _, queueItem := range queueItems {
		if queueItem.EventID != rootEventID {
			otherInputEventID = queueItem.EventID
		}
	}
	require.NotEqual(t, uuid.Nil, otherInputEventID, "fixture sanity: the run must have a second input event")

	// The execution NodeQueueWorker would create for these queue items: it is
	// rooted on the queue items' root event, which is what ListEventExecutions
	// matches on.
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, joinNodeID, rootEventID, queueItems[0].EventID)

	described, err := DescribeRun(context.Background(), database.DB(t.Context()), canvas, response.RunId)
	require.NoError(t, err)
	require.NotNil(t, described.Run.RootEvent)
	assert.Equal(t, rootEventID.String(), described.Run.RootEvent.Id)
	assert.NotEqual(t, otherInputEventID.String(), described.Run.RootEvent.Id,
		"serializing the run's other input event would point at an event nothing is rooted on")

	executions, err := ListEventExecutions(context.Background(), database.DB(t.Context()), canvas, described.Run.RootEvent.Id)
	require.NoError(t, err)
	require.Len(t, executions.Executions, 1, "the run's serialized root event must resolve the run's executions")
	assert.Equal(t, execution.ID.String(), executions.Executions[0].Id)
}
