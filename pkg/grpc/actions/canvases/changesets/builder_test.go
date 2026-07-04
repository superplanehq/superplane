package changesets

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ChangesetBuilder(t *testing.T) {
	support.Setup(t)

	t.Run("no changes", func(t *testing.T) {
		steps := &ChangesetBuilderSteps{t: t}
		steps.whenBuilding(
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

	t.Run("mixed changes", func(t *testing.T) {
		steps := &ChangesetBuilderSteps{t: t}
		steps.whenBuilding(
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
		steps := &ChangesetBuilderSteps{t: t}
		steps.whenBuilding(
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
		steps := &ChangesetBuilderSteps{t: t}
		steps.whenBuilding(
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

	t.Run("add node keeps integration id", func(t *testing.T) {
		steps := &ChangesetBuilderSteps{t: t}
		integrationID := "f453f7c2-a507-41ea-bf5d-8c7482f6dfd4"
		steps.whenBuilding(
			nil,
			nil,
			[]models.Node{
				{
					ID:            "node-a",
					Name:          "Node A",
					Type:          models.NodeTypeComponent,
					IntegrationID: &integrationID,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "github.runWorkflow"},
					},
				},
			},
			nil,
		)

		steps.assertNoError()
		steps.assertOperationCount(1)

		op := steps.findNodeOperation(ChangeTypeAddNode, "node-a")
		require.NotNil(t, op)
		require.Equal(t, integrationID, op.Node.IntegrationID)
	})
}

type ChangesetBuilderSteps struct {
	t         *testing.T
	changeset *CanvasChangeset
	err       error
}

func (s *ChangesetBuilderSteps) whenBuilding(currentNodes []models.Node, currentEdges []models.Edge, proposedNodes []models.Node, proposedEdges []models.Edge) {
	s.changeset, s.err = NewChangesetBuilder(currentNodes, currentEdges, proposedNodes, proposedEdges).Build()
}

func (s *ChangesetBuilderSteps) assertNoError() {
	require.NoError(s.t, s.err)
}

func (s *ChangesetBuilderSteps) assertHasError() {
	require.Error(s.t, s.err)
}

func (s *ChangesetBuilderSteps) assertHasNoOperations() {
	require.Nil(s.t, s.changeset)
}

func (s *ChangesetBuilderSteps) assertOperationCount(count int) {
	require.Len(s.t, s.changeset.Changes, count)
}

func (s *ChangesetBuilderSteps) assertHasDeleteNode(nodeID string) {
	op := s.findNodeOperation(ChangeTypeDeleteNode, nodeID)
	require.NotNil(s.t, op, "expected DELETE_NODE operation for %s", nodeID)
}

func (s *ChangesetBuilderSteps) assertHasAddNode(nodeID string, name string, block string, configuration map[string]any) {
	op := s.findNodeOperation(ChangeTypeAddNode, nodeID)
	require.NotNil(s.t, op, "expected ADD_NODE operation for %s", nodeID)
	require.Equal(s.t, name, op.Node.Name)
	require.Equal(s.t, block, op.Node.Block)
	require.Equal(s.t, configuration, op.Node.Configuration.AsMap())
}

func (s *ChangesetBuilderSteps) assertHasUpdateNode(nodeID string, name string, block string, configuration map[string]any) {
	op := s.findNodeOperation(ChangeTypeUpdateNode, nodeID)
	require.NotNil(s.t, op, "expected UPDATE_NODE operation for %s", nodeID)
	require.Equal(s.t, name, op.Node.Name)
	require.Equal(s.t, block, op.Node.Block)
	require.Equal(s.t, configuration, op.Node.Configuration.AsMap())
}

func (s *ChangesetBuilderSteps) assertHasDisconnect(sourceID string, targetID string, channel string) {
	op := s.findEdgeOperation(ChangeTypeDeleteEdge, sourceID, targetID, channel)
	require.NotNil(s.t, op, "expected DISCONNECT_NODES from %s to %s on channel %s", sourceID, targetID, channel)
}

func (s *ChangesetBuilderSteps) assertHasConnect(sourceID string, targetID string, channel string) {
	op := s.findEdgeOperation(ChangeTypeAddEdge, sourceID, targetID, channel)
	require.NotNil(s.t, op, "expected CONNECT_NODES from %s to %s on channel %s", sourceID, targetID, channel)
}

func (s *ChangesetBuilderSteps) findNodeOperation(operationType ChangeType, nodeID string) *Change {
	for _, change := range s.changeset.Changes {
		if change.Type != operationType {
			continue
		}

		node := change.Node
		if node == nil {
			continue
		}

		if node.ID == nodeID {
			return change
		}
	}

	return nil
}

func (s *ChangesetBuilderSteps) findEdgeOperation(operationType ChangeType, sourceID string, targetID string, channel string) *Change {
	for _, change := range s.changeset.Changes {
		if change.Type != operationType {
			continue
		}

		edge := change.Edge
		if edge == nil {
			continue
		}

		if edge.SourceID != sourceID {
			continue
		}

		if edge.TargetID != targetID {
			continue
		}

		if edge.Channel != channel {
			continue
		}

		return change
	}

	return nil
}
