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
	operations []*pb.CanvasUpdateOperation
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
	op := s.findNodeOperation(pb.CanvasUpdateOperation_DELETE_NODE, nodeID)
	require.NotNil(s.t, op, "expected DELETE_NODE operation for %s", nodeID)
}

func (s *DifferSteps) assertHasAddNode(nodeID string, name string, block string, configuration map[string]any) {
	op := s.findNodeOperation(pb.CanvasUpdateOperation_ADD_NODE, nodeID)
	require.NotNil(s.t, op, "expected ADD_NODE operation for %s", nodeID)
	require.Equal(s.t, name, op.GetTarget().GetName())
	require.Equal(s.t, block, op.GetTarget().GetBlock())
	require.Equal(s.t, configuration, op.GetTarget().GetConfiguration().AsMap())
}

func (s *DifferSteps) assertHasUpdateNode(nodeID string, name string, block string, configuration map[string]any) {
	op := s.findNodeOperation(pb.CanvasUpdateOperation_UPDATE_NODE, nodeID)
	require.NotNil(s.t, op, "expected UPDATE_NODE operation for %s", nodeID)
	require.Equal(s.t, name, op.GetTarget().GetName())
	require.Equal(s.t, block, op.GetTarget().GetBlock())
	require.Equal(s.t, configuration, op.GetTarget().GetConfiguration().AsMap())
}

func (s *DifferSteps) assertHasDisconnect(sourceID string, targetID string, channel string) {
	op := s.findEdgeOperation(pb.CanvasUpdateOperation_DISCONNECT_NODES, sourceID, targetID, channel)
	require.NotNil(s.t, op, "expected DISCONNECT_NODES from %s to %s on channel %s", sourceID, targetID, channel)
}

func (s *DifferSteps) assertHasConnect(sourceID string, targetID string, channel string) {
	op := s.findEdgeOperation(pb.CanvasUpdateOperation_CONNECT_NODES, sourceID, targetID, channel)
	require.NotNil(s.t, op, "expected CONNECT_NODES from %s to %s on channel %s", sourceID, targetID, channel)
}

func (s *DifferSteps) findNodeOperation(operationType pb.CanvasUpdateOperation_Type, nodeID string) *pb.CanvasUpdateOperation {
	for _, operation := range s.operations {
		if operation.GetType() != operationType {
			continue
		}

		target := operation.GetTarget()
		if target == nil {
			continue
		}

		if target.GetId() == nodeID {
			return operation
		}
	}

	return nil
}

func (s *DifferSteps) findEdgeOperation(operationType pb.CanvasUpdateOperation_Type, sourceID string, targetID string, channel string) *pb.CanvasUpdateOperation {
	for _, operation := range s.operations {
		if operation.GetType() != operationType {
			continue
		}

		source := operation.GetSource()
		target := operation.GetTarget()
		if source == nil || target == nil {
			continue
		}

		if source.GetId() != sourceID {
			continue
		}

		if target.GetId() != targetID {
			continue
		}

		if source.GetChannel() != channel || target.GetChannel() != channel {
			continue
		}

		return operation
	}

	return nil
}
