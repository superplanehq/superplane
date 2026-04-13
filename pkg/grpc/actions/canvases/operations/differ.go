package operations

import (
	"fmt"
	"reflect"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
)

/*
 * Differ is responsible for computing the operations to apply
 * to the canvas to go from the current to the proposed version.
 * It makes snapshot-like updates a lot more efficient.
 */
type Differ struct {
	currentNodes  map[string]models.Node
	currentEdges  map[string]models.Edge
	proposedNodes map[string]models.Node
	proposedEdges map[string]models.Edge
}

func NewDiffer(currentNodes []models.Node, currentEdges []models.Edge, proposedNodes []models.Node, proposedEdges []models.Edge) *Differ {
	edgeKeyFn := func(edge models.Edge) string {
		return edge.SourceID + "|" + edge.TargetID + "|" + edge.Channel
	}

	return &Differ{
		currentNodes:  buildNodeMap(currentNodes),
		currentEdges:  buildEdgeMap(currentEdges, edgeKeyFn),
		proposedNodes: buildNodeMap(proposedNodes),
		proposedEdges: buildEdgeMap(proposedEdges, edgeKeyFn),
	}
}

func (d *Differ) Diff() ([]*pb.CanvasUpdateOperation, error) {
	operations := []*pb.CanvasUpdateOperation{}
	operations = append(operations, d.computeDeleteNodeOperations()...)
	operations = append(operations, d.computeAddNodeOperations()...)
	operations = append(operations, d.computeUpdateNodeOperations()...)
	operations = append(operations, d.computeDisconnectNodeOperations()...)
	operations = append(operations, d.computeConnectNodeOperations()...)
	return operations, nil
}

func (d *Differ) computeAddNodeOperations() []*pb.CanvasUpdateOperation {
	operations := []*pb.CanvasUpdateOperation{}

	//
	// If a node exists in the proposed, but not in the current,
	// we need an ADD_NODE operation for it.
	//
	for _, node := range d.proposedNodes {
		if _, ok := d.currentNodes[node.ID]; ok {
			continue
		}

		target, err := nodeToOperationNode(node)
		if err != nil {
			continue
		}

		operations = append(operations, &pb.CanvasUpdateOperation{
			Type:   pb.CanvasUpdateOperation_ADD_NODE,
			Target: target,
		})

		//
		// TODO: do we need to add it to the currentNodes map here too?
		//
	}

	return operations
}

func (d *Differ) computeDeleteNodeOperations() []*pb.CanvasUpdateOperation {
	operations := []*pb.CanvasUpdateOperation{}

	//
	// If a node exists in the current, but not in the proposed,
	// we need a DELETE_NODE operation for it.
	//
	for nodeID := range d.currentNodes {
		if _, ok := d.proposedNodes[nodeID]; ok {
			continue
		}

		operations = append(operations, &pb.CanvasUpdateOperation{
			Type: pb.CanvasUpdateOperation_DELETE_NODE,
			Target: &pb.CanvasUpdateOperation_Node{
				Id: nodeID,
			},
		})
	}

	return operations
}

func (d *Differ) computeUpdateNodeOperations() []*pb.CanvasUpdateOperation {
	operations := []*pb.CanvasUpdateOperation{}

	//
	// If a node exists in both the current and the proposed,
	// but it's not the exactly same node, we need an UPDATE_NODE operation for it.
	//
	for _, node := range d.proposedNodes {
		currentNode, ok := d.currentNodes[node.ID]
		if !ok {
			continue
		}

		if !d.nodeUpdated(currentNode, node) {
			continue
		}

		target, err := nodeToOperationNode(node)
		if err != nil {
			continue
		}

		operations = append(operations, &pb.CanvasUpdateOperation{
			Type:   pb.CanvasUpdateOperation_UPDATE_NODE,
			Target: target,
		})
	}

	return operations
}

func (d *Differ) computeDisconnectNodeOperations() []*pb.CanvasUpdateOperation {
	operations := []*pb.CanvasUpdateOperation{}

	//
	// If an edge exists in the current, but not in the proposed,
	// we need a DISCONNECT_NODES operation for it.
	//
	for edgeKey, edge := range d.currentEdges {
		if _, ok := d.proposedEdges[edgeKey]; ok {
			continue
		}

		operations = append(operations, &pb.CanvasUpdateOperation{
			Type: pb.CanvasUpdateOperation_DISCONNECT_NODES,
			Source: &pb.CanvasUpdateOperation_Node{
				Id:      edge.SourceID,
				Channel: edge.Channel,
			},
			Target: &pb.CanvasUpdateOperation_Node{
				Id:      edge.TargetID,
				Channel: edge.Channel,
			},
		})

		//
		// TODO: do we need to remove it from the currentEdges map here too?
		//
	}

	return operations
}

func (d *Differ) computeConnectNodeOperations() []*pb.CanvasUpdateOperation {
	operations := []*pb.CanvasUpdateOperation{}

	//
	// If an edge exists in the proposed but not in the current,
	// we need a CONNECT_NODES operation.
	//
	for edgeKey, edge := range d.proposedEdges {
		if _, ok := d.currentEdges[edgeKey]; ok {
			continue
		}

		operations = append(operations, &pb.CanvasUpdateOperation{
			Type: pb.CanvasUpdateOperation_CONNECT_NODES,
			Source: &pb.CanvasUpdateOperation_Node{
				Id:      edge.SourceID,
				Channel: edge.Channel,
			},
			Target: &pb.CanvasUpdateOperation_Node{
				Id:      edge.TargetID,
				Channel: edge.Channel,
			},
		})

		//
		// TODO: do we need to add it to the currentEdges map here too?
		//
	}

	return operations
}

func (d *Differ) nodeUpdated(currentNode models.Node, proposedNode models.Node) bool {
	if currentNode.Name != proposedNode.Name {
		return true
	}

	return !reflect.DeepEqual(currentNode.Configuration, proposedNode.Configuration)
}

func nodeToOperationNode(node models.Node) (*pb.CanvasUpdateOperation_Node, error) {
	n := &pb.CanvasUpdateOperation_Node{
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
