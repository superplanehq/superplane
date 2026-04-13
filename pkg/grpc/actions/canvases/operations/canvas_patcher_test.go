package operations

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

func Test__CanvasPatcher(t *testing.T) {
	r := support.Setup(t)

	t.Run("applies mixed operations", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{
				{ID: "node-a", Name: "Node A", Configuration: map[string]any{"foo": "before"}},
				{ID: "node-b", Name: "Node B", Configuration: map[string]any{"bar": "value"}},
			},
			[]models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "default"}},
		)

		steps.whenHandling([]*pb.CanvasUpdateOperation{
			{
				Type: pb.CanvasUpdateOperation_ADD_NODE,
				Target: &pb.CanvasUpdateOperation_Node{
					Id:            "node-c",
					Name:          "Node C",
					Block:         "noop",
					Configuration: structFromMap(t, map[string]any{"baz": "value"}),
				},
			},
			{
				Type: pb.CanvasUpdateOperation_UPDATE_NODE,
				Target: &pb.CanvasUpdateOperation_Node{
					Id:            "node-a",
					Name:          "Node A Updated",
					Configuration: structFromMap(t, map[string]any{"foo": "after"}),
				},
			},
			{
				Type:   pb.CanvasUpdateOperation_DISCONNECT_NODES,
				Source: &pb.CanvasUpdateOperation_Node{Id: "node-a", Channel: "default"},
				Target: &pb.CanvasUpdateOperation_Node{Id: "node-b", Channel: "default"},
			},
			{
				Type:   pb.CanvasUpdateOperation_CONNECT_NODES,
				Source: &pb.CanvasUpdateOperation_Node{Id: "node-a", Channel: "default"},
				Target: &pb.CanvasUpdateOperation_Node{Id: "node-c", Channel: "default"},
			},
			{
				Type:   pb.CanvasUpdateOperation_DELETE_NODE,
				Target: &pb.CanvasUpdateOperation_Node{Id: "node-b"},
			},
		})
		steps.assertNoError()

		steps.assertHasNode("node-a", "Node A Updated", map[string]any{"foo": "after"})
		steps.assertHasNode("node-c", "Node C", map[string]any{"baz": "value"})
		steps.assertHasNodeBlock("node-c", "noop")
		steps.assertNodeCount(2)
		steps.assertHasEdge("node-a", "node-c", "default")
		steps.assertEdgeCount(1)
		steps.assertGraphIsValid()
	})

	t.Run("rejects self-loop edge", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{{ID: "node-a", Name: "Node A"}},
			nil,
		)

		steps.whenHandling([]*pb.CanvasUpdateOperation{
			{
				Type:   pb.CanvasUpdateOperation_CONNECT_NODES,
				Source: &pb.CanvasUpdateOperation_Node{Id: "node-a", Channel: "default"},
				Target: &pb.CanvasUpdateOperation_Node{Id: "node-a", Channel: "default"},
			},
		})
		steps.assertHasError()
	})

	t.Run("rejects unknown operation type", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling([]*pb.CanvasUpdateOperation{
			{
				Type: pb.CanvasUpdateOperation_Type(999),
			},
		})

		steps.assertHasError()
	})

	t.Run("rejects graph with cycles", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{
				{ID: "node-a", Name: "Node A"},
				{ID: "node-b", Name: "Node B"},
			},
			[]models.Edge{
				{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
				{SourceID: "node-b", TargetID: "node-a", Channel: "default"},
			},
		)

		steps.whenHandling([]*pb.CanvasUpdateOperation{})
		steps.assertHasError()
	})
}

type CanvasPatcherSteps struct {
	t        *testing.T
	registry *registry.Registry
	patcher  *CanvasPatcher
	err      error
}

func (s *CanvasPatcherSteps) givenCanvasVersion(nodes []models.Node, edges []models.Edge) {
	s.patcher = NewCanvasPatcher(&models.CanvasVersion{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Nodes:      datatypes.NewJSONSlice(nodes),
		Edges:      datatypes.NewJSONSlice(edges),
	}, s.registry)
}

func (s *CanvasPatcherSteps) whenHandling(operations []*pb.CanvasUpdateOperation) {
	s.err = s.patcher.Patch(operations)
}

func (s *CanvasPatcherSteps) assertNoError() {
	require.NoError(s.t, s.err)
}

func (s *CanvasPatcherSteps) assertHasError() {
	require.Error(s.t, s.err)
}

func (s *CanvasPatcherSteps) assertHasNode(nodeID string, name string, configuration map[string]any) {
	index, found := s.patcher.findNode(nodeID)
	require.True(s.t, found, "expected node %s", nodeID)
	require.Equal(s.t, name, s.patcher.canvas.Nodes[index].Name)
	require.Equal(s.t, configuration, s.patcher.canvas.Nodes[index].Configuration)
}

func (s *CanvasPatcherSteps) assertNodeCount(count int) {
	require.Len(s.t, s.patcher.canvas.Nodes, count)
}

func (s *CanvasPatcherSteps) assertHasNodeBlock(nodeID string, block string) {
	index, found := s.patcher.findNode(nodeID)
	require.True(s.t, found, "expected node %s", nodeID)

	nodeBlock := blockNameFromNode(s.patcher.canvas.Nodes[index])
	require.Equal(s.t, block, nodeBlock)
}

func (s *CanvasPatcherSteps) assertHasEdge(sourceID string, targetID string, channel string) {
	_, found := s.patcher.findEdge(models.Edge{SourceID: sourceID, TargetID: targetID, Channel: channel})
	require.True(s.t, found, "expected edge %s -> %s on channel %s", sourceID, targetID, channel)
}

func (s *CanvasPatcherSteps) assertEdgeCount(count int) {
	require.Len(s.t, s.patcher.canvas.Edges, count)
}

func (s *CanvasPatcherSteps) assertGraphIsValid() {
	require.NoError(s.t, s.patcher.validateCanvasGraph())
}

func structFromMap(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()

	result, err := structpb.NewStruct(value)
	require.NoError(t, err)

	return result
}
