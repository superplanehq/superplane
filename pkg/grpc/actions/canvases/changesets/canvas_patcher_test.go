package changesets

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
				{
					ID:            "node-a",
					Name:          "Node A",
					Configuration: map[string]any{"expression": "true"},
					Type:          models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "if"},
					},
				},
				{
					ID:            "node-b",
					Name:          "Node B",
					Configuration: map[string]any{"expression": "false"},
					Type:          models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "if"},
					},
				},
			},
			[]models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "default"}},
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:            "node-c",
						Name:          "Node C",
						Block:         "noop",
						Configuration: structFromMap(t, map[string]any{}),
					},
				},
				{
					Type: pb.CanvasChangeset_Change_UPDATE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:            "node-a",
						Name:          "Node A Updated",
						Configuration: structFromMap(t, map[string]any{"expression": "false"}),
					},
				},
				{
					Type: pb.CanvasChangeset_Change_ADD_EDGE,
					Edge: &pb.CanvasChangeset_Change_Edge{
						SourceId: "node-a",
						TargetId: "node-c",
						Channel:  "default",
					},
				},
				{
					Type: pb.CanvasChangeset_Change_DELETE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{Id: "node-b"},
				},
			},
		})

		steps.assertNoError()
		steps.assertHasNode("node-a", "Node A Updated", map[string]any{"expression": "false"})
		steps.assertHasNode("node-c", "Node C", map[string]any{})
		steps.assertHasNodeBlock("node-c", "noop")
		steps.assertNodeCount(2)
		steps.assertHasEdge("node-a", "node-c", "default")
		steps.assertEdgeCount(1)
		steps.assertGraphIsValid()
	})

	t.Run("update node -> no configuration provided, previous configuration is preserved", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{{
				ID:            "node-a",
				Name:          "Node A",
				Configuration: map[string]any{"expression": "true"},
				Type:          models.NodeTypeComponent,
				Ref: models.NodeRef{
					Component: &models.ComponentRef{Name: "if"},
				},
			}},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_UPDATE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{Id: "node-a", Name: "Node A Updated"},
				},
			},
		})

		steps.assertNoError()
		steps.assertHasNode("node-a", "Node A Updated", map[string]any{"expression": "true"})
	})

	t.Run("update node -> configuration is validated against block schema", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{
				{
					ID:            "node-a",
					Name:          "Node A",
					Configuration: map[string]any{"expression": "true"},
					Type:          models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "if"},
					},
				},
			},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_UPDATE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:            "node-a",
						Name:          "Node A",
						Configuration: structFromMap(t, map[string]any{"expression": nil}),
					},
				},
			},
		})

		steps.assertHasError()
		steps.assertErrorContains("field 'expression' is required")
	})

	t.Run("rejects self-loop edge", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{{ID: "node-a", Name: "Node A"}},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_EDGE,
					Edge: &pb.CanvasChangeset_Change_Edge{
						SourceId: "node-a",
						TargetId: "node-a",
						Channel:  "default",
					},
				},
			},
		})
		steps.assertHasError()
		steps.assertErrorContains("self-loop edges are not allowed")
	})

	t.Run("rejects block that does not exist", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:    "node-a",
						Name:  "Node A",
						Block: "core.hello",
					},
				},
			},
		})

		steps.assertHasError()
		steps.assertErrorContains("block core.hello not found in registry")
	})

	t.Run("rejects unknown operation type", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_UNSPECIFIED,
				},
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
			},
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_EDGE,
					Edge: &pb.CanvasChangeset_Change_Edge{
						SourceId: "node-b",
						TargetId: "node-a",
						Channel:  "default",
					},
				},
			},
		})
		steps.assertHasError()
	})

	t.Run("rejects add component node when configuration does not match schema", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:    "node-a",
						Name:  "Node A",
						Block: "if",
					},
				},
			},
		})

		steps.assertHasError()
		steps.assertErrorContains("field 'expression' is required")
		steps.assertNodeCount(0)
	})

	t.Run("rejects add trigger node when configuration does not match schema", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:    "node-a",
						Name:  "Node A",
						Block: "schedule",
					},
				},
			},
		})

		steps.assertHasError()
		steps.assertErrorContains("field 'type' is required")
		steps.assertNodeCount(0)
	})

	t.Run("rejects add widget node when configuration does not match schema", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:    "node-a",
						Name:  "Node A",
						Block: "annotation",
					},
				},
			},
		})
		steps.assertHasError()
		steps.assertErrorContains("field 'text' is required")
		steps.assertNodeCount(0)
	})
}

type CanvasPatcherSteps struct {
	t        *testing.T
	registry *registry.Registry
	patcher  *CanvasPatcher
	err      error
}

func (s *CanvasPatcherSteps) givenCanvasVersion(nodes []models.Node, edges []models.Edge) {
	s.patcher = NewCanvasPatcher(s.registry, &models.CanvasVersion{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Nodes:      datatypes.NewJSONSlice(nodes),
		Edges:      datatypes.NewJSONSlice(edges),
	})
}

func (s *CanvasPatcherSteps) whenHandling(operations *pb.CanvasChangeset) {
	s.err = s.patcher.ApplyChangeset(operations)
}

func (s *CanvasPatcherSteps) assertNoError() {
	require.NoError(s.t, s.err)
}

func (s *CanvasPatcherSteps) assertHasError() {
	require.Error(s.t, s.err)
}

func (s *CanvasPatcherSteps) assertErrorContains(text string) {
	require.ErrorContains(s.t, s.err, text)
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

	nodeBlock := s.findBlockName(s.patcher.canvas.Nodes[index])
	require.Equal(s.t, block, nodeBlock)
}

func (s *CanvasPatcherSteps) findBlockName(node models.Node) string {
	if node.Ref.Component != nil && node.Ref.Component.Name != "" {
		return node.Ref.Component.Name
	}

	if node.Ref.Trigger != nil && node.Ref.Trigger.Name != "" {
		return node.Ref.Trigger.Name
	}

	if node.Ref.Widget != nil && node.Ref.Widget.Name != "" {
		return node.Ref.Widget.Name
	}

	return ""
}

func (s *CanvasPatcherSteps) assertHasEdge(sourceID string, targetID string, channel string) {
	_, found := s.patcher.findEdge(sourceID, targetID, channel)
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
