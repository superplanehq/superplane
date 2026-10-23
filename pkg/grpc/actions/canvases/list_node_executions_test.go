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
	"gorm.io/datatypes"
)

func Test__ListNodeExecutions(t *testing.T) {
	r := support.Setup(t)

	t.Run("node does not exist -> 404 error", func(t *testing.T) {
		//
		// Create a canvas with a node
		//
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Try to list executions for a non-existent node
		//
		_, err := ListNodeExecutions(
			context.Background(),
			database.DB(t.Context()),
			canvas,
			"non-existent-node",
			[]pb.CanvasNodeExecution_State{},
			[]pb.CanvasNodeExecution_Result{},
			0,
			nil,
		)

		//
		// Verify we get a NotFound error
		//
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
		assert.Contains(t, msg, "canvas node not found")
	})

	t.Run("returns executions for existing node", func(t *testing.T) {
		//
		// Create a canvas with a node
		//
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Create events and executions
		//
		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		customName := "Custom Root Event"
		rootEvent.CustomName = &customName
		require.NoError(t, database.Conn().Save(rootEvent).Error)
		event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, event.ID)

		//
		// List executions for the node
		//
		response, err := ListNodeExecutions(
			context.Background(),
			database.DB(t.Context()),
			canvas,
			"node-1",
			[]pb.CanvasNodeExecution_State{},
			[]pb.CanvasNodeExecution_Result{},
			0,
			nil,
		)

		//
		// Verify successful response
		//
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Len(t, response.Executions, 1)
		assert.Equal(t, uint32(1), response.TotalCount)
		assert.Equal(t, "node-1", response.Executions[0].NodeId)
		assert.Equal(t, rootEvent.RunID.String(), response.Executions[0].RunId)
		require.NotNil(t, response.Executions[0].RootEvent)
		assert.Equal(t, customName, response.Executions[0].RootEvent.CustomName)
		assert.Equal(t, rootEvent.RunID.String(), response.Executions[0].RootEvent.RunId)
	})

	t.Run("execution from a single input event carries input_event with that event's id and data", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
				},
			},
			[]models.Edge{},
		)

		event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "node-1", "default", nil, map[string]any{"foo": "bar"})
		support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event.ID, event.ID)

		response, err := ListNodeExecutions(
			context.Background(),
			database.DB(t.Context()),
			canvas,
			"node-1",
			[]pb.CanvasNodeExecution_State{},
			[]pb.CanvasNodeExecution_Result{},
			0,
			nil,
		)

		require.NoError(t, err)
		require.Len(t, response.Executions, 1)
		require.NotNil(t, response.Executions[0].InputEvent)
		assert.Equal(t, event.ID.String(), response.Executions[0].InputEvent.Id)
		assert.Equal(t, "bar", response.Executions[0].InputEvent.Data.AsMap()["foo"])
	})

	t.Run("mid-chain execution has input_event pointing at the upstream output event, distinct from root_event", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: "node-2",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
				},
			},
			[]models.Edge{},
		)

		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
		upstreamExecutionID := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID).ID
		upstreamOutputEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "node-1", "default", &upstreamExecutionID, map[string]any{"stage": "one"})
		support.CreateCanvasNodeExecution(t, canvas.ID, "node-2", rootEvent.ID, upstreamOutputEvent.ID)

		response, err := ListNodeExecutions(
			context.Background(),
			database.DB(t.Context()),
			canvas,
			"node-2",
			[]pb.CanvasNodeExecution_State{},
			[]pb.CanvasNodeExecution_Result{},
			0,
			nil,
		)

		require.NoError(t, err)
		require.Len(t, response.Executions, 1)
		require.NotNil(t, response.Executions[0].InputEvent)
		require.NotNil(t, response.Executions[0].RootEvent)
		assert.Equal(t, upstreamOutputEvent.ID.String(), response.Executions[0].InputEvent.Id)
		assert.Equal(t, rootEvent.ID.String(), response.Executions[0].RootEvent.Id)
		assert.NotEqual(t, response.Executions[0].RootEvent.Id, response.Executions[0].InputEvent.Id)
	})

	t.Run("execution created by a configuration build error has input_event and does not fail the response", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
				},
			},
			[]models.Edge{},
		)

		triggeringEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", triggeringEvent.ID, triggeringEvent.ID)
		execution.State = models.CanvasNodeExecutionStateFinished
		execution.Result = models.CanvasNodeExecutionResultFailed
		execution.ResultReason = models.CanvasNodeExecutionResultReasonError
		execution.ResultMessage = "error building configuration"
		require.NoError(t, database.Conn().Save(execution).Error)

		response, err := ListNodeExecutions(
			context.Background(),
			database.DB(t.Context()),
			canvas,
			"node-1",
			[]pb.CanvasNodeExecution_State{},
			[]pb.CanvasNodeExecution_Result{},
			0,
			nil,
		)

		require.NoError(t, err)
		require.Len(t, response.Executions, 1)
		require.NotNil(t, response.Executions[0].InputEvent)
		assert.Equal(t, triggeringEvent.ID.String(), response.Executions[0].InputEvent.Id)
	})

	t.Run("execution of a replay run carries is_replay and the source execution it was replayed from", func(t *testing.T) {
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

		originalEvent := support.EmitCanvasEventForNode(t, canvas.ID, sourceNodeID, "default", nil)
		originalExecution := support.CreateCanvasNodeExecution(t, canvas.ID, targetNodeID, originalEvent.ID, originalEvent.ID)
		require.NoError(t, models.CreateConsumedEvent(database.Conn(), originalExecution.ID, originalEvent.ID))

		replayResponse, err := ReplayNode(context.Background(), database.DB(t.Context()), canvas, targetNodeID, &originalExecution.ID, nil, nil, nil)
		require.NoError(t, err)
		require.Len(t, replayResponse.QueueItemIds, 1)

		queueItemID, err := uuid.Parse(replayResponse.QueueItemIds[0])
		require.NoError(t, err)
		queueItem, err := models.FindNodeQueueItem(canvas.ID, queueItemID)
		require.NoError(t, err)

		replayExecution := support.CreateCanvasNodeExecution(t, canvas.ID, targetNodeID, queueItem.RootEventID, queueItem.EventID)
		require.Equal(t, replayResponse.RunId, replayExecution.RunID.String(), "fixture sanity: the execution must belong to the replay run")

		response, err := ListNodeExecutions(
			context.Background(),
			database.DB(t.Context()),
			canvas,
			targetNodeID,
			[]pb.CanvasNodeExecution_State{},
			[]pb.CanvasNodeExecution_Result{},
			0,
			nil,
		)
		require.NoError(t, err)
		require.Len(t, response.Executions, 2)

		serializedByID := map[string]*pb.CanvasNodeExecution{}
		for _, execution := range response.Executions {
			serializedByID[execution.Id] = execution
		}

		serializedReplay, ok := serializedByID[replayExecution.ID.String()]
		require.True(t, ok)
		assert.True(t, serializedReplay.IsReplay, "an execution of a replay run must be serialized as a replay")
		assert.Equal(t, originalExecution.ID.String(), serializedReplay.ReplaySourceExecutionId)

		serializedOriginal, ok := serializedByID[originalExecution.ID.String()]
		require.True(t, ok)
		assert.False(t, serializedOriginal.IsReplay, "the replayed execution itself is not a replay")
		assert.Empty(t, serializedOriginal.ReplaySourceExecutionId)
	})

	t.Run("execution whose input event was deleted by retention has empty input_event and response succeeds", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: "node-2",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
				},
			},
			[]models.Edge{},
		)

		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)
		inputEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		support.CreateCanvasNodeExecution(t, canvas.ID, "node-2", rootEvent.ID, inputEvent.ID)

		//
		// Simulate retention deleting the input event. The FK from
		// workflow_node_executions.event_id is ON DELETE SET NULL, so the
		// execution row survives with a nil EventID.
		//
		require.NoError(t, database.Conn().Where("id = ?", inputEvent.ID).Delete(&models.CanvasEvent{}).Error)

		response, err := ListNodeExecutions(
			context.Background(),
			database.DB(t.Context()),
			canvas,
			"node-2",
			[]pb.CanvasNodeExecution_State{},
			[]pb.CanvasNodeExecution_Result{},
			0,
			nil,
		)

		require.NoError(t, err)
		require.Len(t, response.Executions, 1)
		assert.Nil(t, response.Executions[0].InputEvent)
		require.NotNil(t, response.Executions[0].RootEvent)
		assert.Equal(t, rootEvent.ID.String(), response.Executions[0].RootEvent.Id)
	})
}

// workflow_node_executions.run_id is NOT NULL with a foreign key, so neither
// state below can be built through the database - only by calling the
// serializer's helper directly.
func Test__ReplayLineageForExecution_WithoutALoadedRun(t *testing.T) {
	replayRun := models.CanvasRun{ID: uuid.New(), IsReplay: true}

	t.Run("execution that belongs to no run is not a replay", func(t *testing.T) {
		isReplay, sourceExecutionID := replayLineageForExecution(
			models.CanvasNodeExecution{RunID: uuid.Nil},
			map[uuid.UUID]models.CanvasRun{replayRun.ID: replayRun},
		)

		assert.False(t, isReplay)
		assert.Empty(t, sourceExecutionID)
	})

	t.Run("execution whose run was not loaded is not a replay", func(t *testing.T) {
		isReplay, sourceExecutionID := replayLineageForExecution(
			models.CanvasNodeExecution{RunID: uuid.New()},
			map[uuid.UUID]models.CanvasRun{replayRun.ID: replayRun},
		)

		assert.False(t, isReplay)
		assert.Empty(t, sourceExecutionID)
	})
}

func SerializeThosandNodeExecutions(b *testing.B) {
	r := support.Setup(b)

	//
	// Create a simple canvas with single trigger and component
	//
	canvas, _ := support.CreateCanvas(b, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "manual",
			Name:   "Manual start",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
		},
		{
			NodeID: "node-1",
			Name:   "Test Node",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			}),
		},
	}, []models.Edge{})

	//
	// Generate 1000 executions for the component node
	//
	for i := 0; i < 1000; i++ {
		event := support.EmitCanvasEventForNode(b, canvas.ID, "manual", "default", nil)
		execution := support.CreateCanvasNodeExecution(b, canvas.ID, "node-1", event.ID, event.ID)
		_, err := execution.Pass(map[string][]any{"default": {map[string]any{"data": "test"}}})
		require.NoError(b, err)
	}

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, "node-1", []string{}, []string{}, 1000, nil)
	require.NoError(b, err)

	resources, err := LoadNodeExecutionResources(database.Conn(), executions)
	require.NoError(b, err)

	b.ResetTimer()
	for b.Loop() {
		pb, err := SerializeNodeExecutions(executions, resources)
		require.NoError(b, err)
		require.NotNil(b, pb)
		assert.Len(b, pb, 1000)
	}
}

func Test__BenchmarkSerializeNodeExecutions(t *testing.T) {
	//
	// Serializing 1000 executions should take no longer than 50ms
	//
	res := testing.Benchmark(SerializeThosandNodeExecutions)
	assert.Less(t, res.NsPerOp(), int64(50000000))
}
