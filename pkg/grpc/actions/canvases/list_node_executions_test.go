package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func Test__ListNodeExecutions(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := ListNodeExecutions(
			context.Background(),
			r.Registry,
			"invalid-uuid",
			"some-node",
			[]pb.CanvasNodeExecution_State{},
			[]pb.CanvasNodeExecution_Result{},
			0,
			nil,
		)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

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
			r.Registry,
			canvas.ID.String(),
			"non-existent-node",
			[]pb.CanvasNodeExecution_State{},
			[]pb.CanvasNodeExecution_Result{},
			0,
			nil,
		)

		//
		// Verify we get a NotFound error
		//
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "canvas node not found")
	})

	t.Run("canvas does not exist -> 404 error", func(t *testing.T) {
		//
		// Try to list executions for a non-existent canvas
		//
		_, err := ListNodeExecutions(
			context.Background(),
			r.Registry,
			uuid.New().String(),
			"some-node",
			[]pb.CanvasNodeExecution_State{},
			[]pb.CanvasNodeExecution_Result{},
			0,
			nil,
		)

		//
		// Verify we get a NotFound error
		//
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "canvas node not found")
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
		support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, event.ID, nil)

		//
		// List executions for the node
		//
		response, err := ListNodeExecutions(
			context.Background(),
			r.Registry,
			canvas.ID.String(),
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
		require.NotNil(t, response.Executions[0].RootEvent)
		assert.Equal(t, customName, response.Executions[0].RootEvent.CustomName)
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
		execution := support.CreateCanvasNodeExecution(b, canvas.ID, "node-1", event.ID, event.ID, nil)
		_, err := execution.Pass(map[string][]any{"default": {map[string]any{"data": "test"}}})
		require.NoError(b, err)
	}

	executions, err := models.ListNodeExecutions(canvas.ID, "node-1", []string{}, []string{}, 1000, nil)
	require.NoError(b, err)

	b.ResetTimer()
	for b.Loop() {
		pb, err := SerializeNodeExecutions(executions, []models.CanvasNodeExecution{})
		require.NoError(b, err)
		require.NotNil(b, pb)
		assert.Len(b, pb, 1000)
	}
}

func Test__BenchmarkSerializeNodeExecutions(t *testing.T) {
	//
	// Serializing 1000 executions should take no longer than 10ms
	//
	res := testing.Benchmark(SerializeThosandNodeExecutions)
	assert.Less(t, res.NsPerOp(), int64(10000000))
}
