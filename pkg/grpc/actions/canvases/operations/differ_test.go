package operations

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func Test__Differ(t *testing.T) {
	support.Setup(t)

	t.Run("no changes", func(t *testing.T) {
		steps := &DifferSteps{t: t}
		steps.whenDiffing(
			[]models.Node{
				{
					ID:   "node-a",
					Name: "Node A",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"foo": "bar"},
				},
				{
					ID:   "node-b",
					Name: "Node B",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"baz": "qux"},
				},
			},
			[]models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "default"}},
			[]models.Node{
				{
					ID:   "node-a",
					Name: "Node A",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"foo": "bar"},
				},
				{
					ID:   "node-b",
					Name: "Node B",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"baz": "qux"},
				},
			},
			[]models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "default"}},
		)

		steps.assertNoError()
		steps.assertOperationCount(0)
	})

	t.Run("mixed operations", func(t *testing.T) {
		steps := &DifferSteps{t: t}
		steps.whenDiffing(
			[]models.Node{
				{
					ID:   "node-a",
					Name: "Node A",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"foo": "before"},
				},
				{
					ID:   "node-b",
					Name: "Node B",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"bar": "value"},
				},
			},
			[]models.Edge{
				{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
			},
			[]models.Node{
				{
					ID:   "node-a",
					Name: "Node A Updated",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"foo": "after"},
				},
				{
					ID:   "node-c",
					Name: "Node C",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"baz": "value"},
				},
			},
			[]models.Edge{
				{SourceID: "node-a", TargetID: "node-c", Channel: "default"},
			},
		)

		steps.assertNoError()
		steps.assertOperationCount(5)
		steps.assertHasDeleteNode("node-b")
		steps.assertHasAddNode("node-c", "Node C", "noop", map[string]any{"baz": "value"})
		steps.assertHasUpdateNode("node-a", "Node A Updated", "noop", map[string]any{"foo": "after"})
		steps.assertHasDisconnect("node-a", "node-b", "default")
		steps.assertHasConnect("node-a", "node-c", "default")
	})

	t.Run("invalid configuration for added node returns error", func(t *testing.T) {
		steps := &DifferSteps{t: t}
		steps.whenDiffing(
			nil,
			nil,
			[]models.Node{
				{
					ID:   "node-a",
					Name: "Node A",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"invalid": func() {}},
				},
			},
			nil,
		)

		steps.assertHasError()
		steps.assertHasNoOperations()
	})

	t.Run("invalid configuration for updated node returns error", func(t *testing.T) {
		steps := &DifferSteps{t: t}
		steps.whenDiffing(
			[]models.Node{
				{
					ID:   "node-a",
					Name: "Node A",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"valid": "value"},
				},
			},
			nil,
			[]models.Node{
				{
					ID:   "node-a",
					Name: "Node A",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Configuration: map[string]any{"invalid": func() {}},
				},
			},
			nil,
		)

		steps.assertHasError()
		steps.assertHasNoOperations()
	})
}

type DifferSteps struct {
	t          *testing.T
	operations []*pb.PatchOperation
	err        error
}

func (s *DifferSteps) whenDiffing(currentNodes []models.Node, currentEdges []models.Edge, proposedNodes []models.Node, proposedEdges []models.Edge) {
	s.operations, s.err = NewDiffer(currentNodes, currentEdges, proposedNodes, proposedEdges).Diff()
}

func (s *DifferSteps) assertNoError() {
	require.NoError(s.t, s.err)
}

func (s *DifferSteps) assertHasError() {
	require.Error(s.t, s.err)
}

func (s *DifferSteps) assertHasNoOperations() {
	require.Nil(s.t, s.operations)
}

func (s *DifferSteps) assertOperationCount(count int) {
	require.Len(s.t, s.operations, count)
}

func (s *DifferSteps) assertHasDeleteNode(nodeID string) {
	op := s.findNodeOperation(pb.PatchOperation_DELETE_NODE, nodeID)
	require.NotNil(s.t, op, "expected DELETE_NODE operation for %s", nodeID)
}

func (s *DifferSteps) assertHasAddNode(nodeID string, name string, block string, configuration map[string]any) {
	op := s.findNodeOperation(pb.PatchOperation_ADD_NODE, nodeID)
	require.NotNil(s.t, op, "expected ADD_NODE operation for %s", nodeID)
	require.Equal(s.t, name, op.GetNode().GetName())
	require.Equal(s.t, block, op.GetNode().GetBlock())
	require.Equal(s.t, configuration, op.GetNode().GetConfiguration().AsMap())
}

func (s *DifferSteps) assertHasUpdateNode(nodeID string, name string, block string, configuration map[string]any) {
	op := s.findNodeOperation(pb.PatchOperation_UPDATE_NODE, nodeID)
	require.NotNil(s.t, op, "expected UPDATE_NODE operation for %s", nodeID)
	require.Equal(s.t, name, op.GetNode().GetName())
	require.Equal(s.t, block, op.GetNode().GetBlock())
	require.Equal(s.t, configuration, op.GetNode().GetConfiguration().AsMap())
}

func (s *DifferSteps) assertHasDisconnect(sourceID string, targetID string, channel string) {
	op := s.findEdgeOperation(pb.PatchOperation_DISCONNECT_NODES, sourceID, targetID, channel)
	require.NotNil(s.t, op, "expected DISCONNECT_NODES from %s to %s on channel %s", sourceID, targetID, channel)
}

func (s *DifferSteps) assertHasConnect(sourceID string, targetID string, channel string) {
	op := s.findEdgeOperation(pb.PatchOperation_CONNECT_NODES, sourceID, targetID, channel)
	require.NotNil(s.t, op, "expected CONNECT_NODES from %s to %s on channel %s", sourceID, targetID, channel)
}

func (s *DifferSteps) findNodeOperation(operationType pb.PatchOperation_Type, nodeID string) *pb.PatchOperation {
	for _, operation := range s.operations {
		if operation.GetType() != operationType {
			continue
		}

		node := operation.GetNode()
		if node == nil {
			continue
		}

		if node.GetId() == nodeID {
			return operation
		}
	}

	return nil
}

func (s *DifferSteps) findEdgeOperation(operationType pb.PatchOperation_Type, sourceID string, targetID string, channel string) *pb.PatchOperation {
	for _, operation := range s.operations {
		if operation.GetType() != operationType {
			continue
		}

		edge := operation.GetEdge()
		if edge == nil {
			continue
		}

		if edge.GetSourceId() != sourceID {
			continue
		}

		if edge.GetTargetId() != targetID {
			continue
		}

		if edge.GetChannel() != channel {
			continue
		}

		return operation
	}

	return nil
}
