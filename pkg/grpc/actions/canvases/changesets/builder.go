package changesets

import (
	"fmt"
	"reflect"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

/*
 * ChangesetBuilder is responsible for computing the operations to apply
 * to the canvas to go from the current to the proposed version.
 * It makes snapshot-like updates a lot more efficient.
 */
type ChangesetBuilder struct {
	currentNodes  map[string]models.Node
	currentEdges  map[string]models.Edge
	proposedNodes map[string]models.Node
	proposedEdges map[string]models.Edge
}

func NewChangesetBuilder(currentNodes []models.Node, currentEdges []models.Edge, proposedNodes []models.Node, proposedEdges []models.Edge) *ChangesetBuilder {
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

		node, err := changeNodeRefForAdd(node)
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

		if reflect.DeepEqual(currentNode, node) {
			continue
		}

		n, err := changeNodeRefForUpdate(currentNode, node)
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

func changeNodeRefForAdd(proposedNode models.Node) (*pb.CanvasChangeset_Change_Node, error) {
	n := &pb.CanvasChangeset_Change_Node{
		Id:          proposedNode.ID,
		Name:        proposedNode.Name,
		Block:       blockNameFromNode(proposedNode),
		IsCollapsed: proto.Bool(proposedNode.IsCollapsed),
		Position: &componentpb.Position{
			X: int32(proposedNode.Position.X),
			Y: int32(proposedNode.Position.Y),
		},
	}

	if proposedNode.IntegrationID != nil {
		n.IntegrationId = *proposedNode.IntegrationID
	}

	if proposedNode.Configuration != nil {
		configuration, err := structpb.NewStruct(proposedNode.Configuration)
		if err != nil {
			return nil, fmt.Errorf("invalid configuration for node %s: %w", proposedNode.ID, err)
		}

		n.Configuration = configuration
	}

	return n, nil
}

func changeNodeRefForUpdate(currentNode models.Node, proposedNode models.Node) (*pb.CanvasChangeset_Change_Node, error) {
	n := &pb.CanvasChangeset_Change_Node{
		Id:    proposedNode.ID,
		Name:  proposedNode.Name,
		Block: blockNameFromNode(proposedNode),
	}

	//
	// If the configuration is different, we set configuration in the change.
	//
	if proposedNode.Configuration != nil && !reflect.DeepEqual(currentNode.Configuration, proposedNode.Configuration) {
		configuration, err := structpb.NewStruct(proposedNode.Configuration)
		if err != nil {
			return nil, fmt.Errorf("invalid configuration for node %s: %w", proposedNode.ID, err)
		}

		n.Configuration = configuration
	}

	//
	// If the position is different, we set position in the change.
	//
	if proposedNode.Position.X != currentNode.Position.X || proposedNode.Position.Y != currentNode.Position.Y {
		n.Position = &componentpb.Position{
			X: int32(proposedNode.Position.X),
			Y: int32(proposedNode.Position.Y),
		}
	}

	//
	// If the collapsed state is different, we set it in the change.
	//
	if proposedNode.IsCollapsed != currentNode.IsCollapsed {
		n.IsCollapsed = proto.Bool(proposedNode.IsCollapsed)
	}

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

func edgeKeyFn(edge models.Edge) string {
	return edge.SourceID + "|" + edge.TargetID + "|" + edge.Channel
}
