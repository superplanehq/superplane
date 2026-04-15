package changesets

import (
	"fmt"
	"reflect"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
)

/*
 * ChangesetBuilder computes a changeset
 * to go from the current version to the proposed version.
 * This helps us avoid applying unnecessary changes to the
 * canvas as part of a canvas version update / publish.
 */
type ChangesetBuilder struct {
	currentNodes  map[string]models.Node
	currentEdges  map[string]models.Edge
	proposedNodes map[string]models.Node
	proposedEdges map[string]models.Edge
}

func NewChangesetBuilder(currentNodes []models.Node, currentEdges []models.Edge, proposedNodes []models.Node, proposedEdges []models.Edge) *ChangesetBuilder {
	edgeKeyFn := func(edge models.Edge) string {
		return edge.SourceID + "|" + edge.TargetID + "|" + edge.Channel
	}

	return &ChangesetBuilder{
		currentNodes:  buildNodeMap(currentNodes),
		currentEdges:  buildEdgeMap(currentEdges, edgeKeyFn),
		proposedNodes: buildNodeMap(proposedNodes),
		proposedEdges: buildEdgeMap(proposedEdges, edgeKeyFn),
	}
}

func (b *ChangesetBuilder) Build() (*pb.CanvasChangeset, error) {
	allChanges := []*pb.CanvasChangeset_Change{}
	allChanges = append(allChanges, b.computeDeleteNodeChanges()...)
	changes, err := b.computeAddNodeChanges()
	if err != nil {
		return nil, err
	}

	allChanges = append(allChanges, changes...)

	changes, err = b.computeUpdateNodeChanges()
	if err != nil {
		return nil, err
	}

	allChanges = append(allChanges, changes...)
	allChanges = append(allChanges, b.computeDisconnectNodeChanges()...)
	allChanges = append(allChanges, b.computeConnectNodeChanges()...)
	return &pb.CanvasChangeset{Changes: allChanges}, nil
}

func (b *ChangesetBuilder) computeAddNodeChanges() ([]*pb.CanvasChangeset_Change, error) {
	changes := []*pb.CanvasChangeset_Change{}

	//
	// If a node exists in the proposed, but not in the current,
	// we need an ADD_NODE operation for it.
	//
	for _, node := range b.proposedNodes {
		if _, ok := b.currentNodes[node.ID]; ok {
			continue
		}

		node, err := nodeToOperationNode(node)
		if err != nil {
			return nil, err
		}

		changes = append(changes, &pb.CanvasChangeset_Change{
			Type: pb.CanvasChangeset_Change_ADD_NODE,
			Node: node,
		})
	}

	return changes, nil
}

func (b *ChangesetBuilder) computeDeleteNodeChanges() []*pb.CanvasChangeset_Change {
	changes := []*pb.CanvasChangeset_Change{}

	//
	// If a node exists in the current, but not in the proposed,
	// we need a DELETE_NODE operation for it.
	//
	for nodeID := range b.currentNodes {
		if _, ok := b.proposedNodes[nodeID]; ok {
			continue
		}

		changes = append(changes, &pb.CanvasChangeset_Change{
			Type: pb.CanvasChangeset_Change_DELETE_NODE,
			Node: &pb.CanvasChangeset_Change_Node{
				Id: nodeID,
			},
		})
	}

	return changes
}

func (b *ChangesetBuilder) computeUpdateNodeChanges() ([]*pb.CanvasChangeset_Change, error) {
	changes := []*pb.CanvasChangeset_Change{}

	//
	// If a node exists in both the current and the proposed,
	// but it's not the exactly same node, we need an UPDATE_NODE operation for it.
	//
	for _, node := range b.proposedNodes {
		currentNode, ok := b.currentNodes[node.ID]
		if !ok {
			continue
		}

		if !b.nodeUpdated(currentNode, node) {
			continue
		}

		n, err := nodeToOperationNode(node)
		if err != nil {
			return nil, err
		}

		changes = append(changes, &pb.CanvasChangeset_Change{
			Type: pb.CanvasChangeset_Change_UPDATE_NODE,
			Node: n,
		})
	}

	return changes, nil
}

func (b *ChangesetBuilder) computeDisconnectNodeChanges() []*pb.CanvasChangeset_Change {
	changes := []*pb.CanvasChangeset_Change{}

	//
	// If an edge exists in the current, but not in the proposed,
	// we need a DISCONNECT_NODES operation for it.
	//
	for edgeKey, edge := range b.currentEdges {
		if _, ok := b.proposedEdges[edgeKey]; ok {
			continue
		}

		changes = append(changes, &pb.CanvasChangeset_Change{
			Type: pb.CanvasChangeset_Change_DELETE_EDGE,
			Edge: &pb.CanvasChangeset_Change_Edge{
				SourceId: edge.SourceID,
				TargetId: edge.TargetID,
				Channel:  edge.Channel,
			},
		})
	}

	return changes
}

func (b *ChangesetBuilder) computeConnectNodeChanges() []*pb.CanvasChangeset_Change {
	changes := []*pb.CanvasChangeset_Change{}

	//
	// If an edge exists in the proposed but not in the current,
	// we need a CONNECT_NODES operation.
	//
	for edgeKey, edge := range b.proposedEdges {
		if _, ok := b.currentEdges[edgeKey]; ok {
			continue
		}

		changes = append(changes, &pb.CanvasChangeset_Change{
			Type: pb.CanvasChangeset_Change_ADD_EDGE,
			Edge: &pb.CanvasChangeset_Change_Edge{
				SourceId: edge.SourceID,
				TargetId: edge.TargetID,
				Channel:  edge.Channel,
			},
		})
	}

	return changes
}

func (b *ChangesetBuilder) nodeUpdated(currentNode models.Node, proposedNode models.Node) bool {
	if currentNode.Name != proposedNode.Name {
		return true
	}

	return !reflect.DeepEqual(currentNode.Configuration, proposedNode.Configuration)
}

func blockNameFromNode(node models.Node) string {
	if node.Ref.Component != nil && node.Ref.Component.Name != "" {
		return node.Ref.Component.Name
	}

	if node.Ref.Trigger != nil && node.Ref.Trigger.Name != "" {
		return node.Ref.Trigger.Name
	}

	if node.Ref.Blueprint != nil && node.Ref.Blueprint.ID != "" {
		return node.Ref.Blueprint.ID
	}

	if node.Ref.Widget != nil && node.Ref.Widget.Name != "" {
		return node.Ref.Widget.Name
	}

	return ""
}

func nodeToOperationNode(node models.Node) (*pb.CanvasChangeset_Change_Node, error) {
	n := &pb.CanvasChangeset_Change_Node{
		Id:   node.ID,
		Name: node.Name,
	}

	if node.Configuration != nil {
		configuration, err := structpb.NewStruct(node.Configuration)
		if err != nil {
			return nil, fmt.Errorf("invalid configuration for node %s: %w", node.ID, err)
		}

		n.Configuration = configuration
	}

	blockName := blockNameFromNode(node)
	if blockName == "" {
		return nil, fmt.Errorf("block name is required for node %s", node.ID)
	}

	n.Block = blockName
	return n, nil
}

func buildNodeMap(nodes []models.Node) map[string]models.Node {
	m := make(map[string]models.Node, len(nodes))
	for _, node := range nodes {
		if node.ID == "" {
			continue
		}

		m[node.ID] = node
	}

	return m
}

func buildEdgeMap(edges []models.Edge, keyFn func(models.Edge) string) map[string]models.Edge {
	m := make(map[string]models.Edge, len(edges))
	for _, edge := range edges {
		m[keyFn(edge)] = edge
	}

	return m
}
