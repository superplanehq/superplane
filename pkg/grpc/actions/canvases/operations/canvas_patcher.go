package operations

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
)

type CanvasPatcher struct {
	canvas   *models.CanvasVersion
	registry *registry.Registry
}

func NewCanvasPatcher(canvas *models.CanvasVersion, registry *registry.Registry) *CanvasPatcher {
	return &CanvasPatcher{
		canvas:   canvas,
		registry: registry,
	}
}

func (p *CanvasPatcher) GetVersion() *models.CanvasVersion {
	return p.canvas
}

func (p *CanvasPatcher) Patch(operations []*pb.CanvasUpdateOperation) error {
	for _, operation := range operations {
		err := p.handleOperation(operation)
		if err != nil {
			return err
		}
	}

	return p.validateCanvasGraph()
}

func (p *CanvasPatcher) handleOperation(operation *pb.CanvasUpdateOperation) error {
	if operation == nil {
		return fmt.Errorf("operation is required")
	}

	switch operation.Type {
	case pb.CanvasUpdateOperation_ADD_NODE:
		return p.addNode(operation)
	case pb.CanvasUpdateOperation_DELETE_NODE:
		return p.deleteNode(operation)
	case pb.CanvasUpdateOperation_UPDATE_NODE:
		return p.updateNode(operation)
	case pb.CanvasUpdateOperation_CONNECT_NODES:
		return p.connectNodes(operation)
	case pb.CanvasUpdateOperation_DISCONNECT_NODES:
		return p.disconnectNodes(operation)
	}

	return fmt.Errorf("unknown operation type: %s", operation.Type)
}

func (p *CanvasPatcher) addNode(operation *pb.CanvasUpdateOperation) error {
	target := operation.GetTarget()
	if target == nil {
		return fmt.Errorf("target is required for %s", operation.Type)
	}

	nodeID := target.GetId()
	if nodeID == "" {
		return fmt.Errorf("target node id is required for %s", operation.Type)
	}

	if target.GetName() == "" {
		return fmt.Errorf("target node name is required for %s", operation.Type)
	}

	if _, found := p.findNode(nodeID); found {
		return fmt.Errorf("node %s already exists", nodeID)
	}

	nodeType, nodeRef, err := p.findBlock(target.GetBlock())
	if err != nil {
		return err
	}

	var configuration map[string]any
	if target.GetConfiguration() != nil {
		configuration = target.GetConfiguration().AsMap()
	}

	p.canvas.Nodes = append(p.canvas.Nodes, models.Node{
		ID:            nodeID,
		Name:          target.GetName(),
		Type:          nodeType,
		Ref:           *nodeRef,
		Configuration: configuration,
	})

	return nil
}

func (p *CanvasPatcher) deleteNode(operation *pb.CanvasUpdateOperation) error {
	target := operation.GetTarget()
	if target == nil {
		return fmt.Errorf("target is required for %s", operation.Type)
	}

	nodeID := target.GetId()
	if nodeID == "" {
		return fmt.Errorf("target node id is required for %s", operation.Type)
	}

	nodeIndex, found := p.findNode(nodeID)
	if !found {
		return fmt.Errorf("node %s not found", nodeID)
	}

	p.canvas.Nodes = slices.Delete(p.canvas.Nodes, nodeIndex, nodeIndex+1)
	p.canvas.Edges = slices.DeleteFunc(p.canvas.Edges, func(edge models.Edge) bool {
		return edge.SourceID == nodeID || edge.TargetID == nodeID
	})

	return nil
}

func (p *CanvasPatcher) updateNode(operation *pb.CanvasUpdateOperation) error {
	target := operation.GetTarget()
	if target == nil {
		return fmt.Errorf("target is required for %s", operation.Type)
	}

	nodeID := target.GetId()
	if nodeID == "" {
		return fmt.Errorf("target node id is required for %s", operation.Type)
	}

	if target.GetName() == "" {
		return fmt.Errorf("target node name is required for %s", operation.Type)
	}

	nodeIndex, found := p.findNode(nodeID)
	if !found {
		return fmt.Errorf("node %s not found", nodeID)
	}

	var configuration map[string]any
	if target.GetConfiguration() != nil {
		configuration = target.GetConfiguration().AsMap()
	}

	p.canvas.Nodes[nodeIndex].Name = target.GetName()
	p.canvas.Nodes[nodeIndex].Configuration = configuration

	return nil
}

func (p *CanvasPatcher) connectNodes(operation *pb.CanvasUpdateOperation) error {
	edge, err := edgeFromOperation(operation)
	if err != nil {
		return err
	}

	if edge.SourceID == edge.TargetID {
		return fmt.Errorf("self-loop edges are not allowed")
	}

	if _, found := p.findNode(edge.SourceID); !found {
		return fmt.Errorf("source node %s not found", edge.SourceID)
	}

	if _, found := p.findNode(edge.TargetID); !found {
		return fmt.Errorf("target node %s not found", edge.TargetID)
	}

	if _, found := p.findEdge(edge); found {
		return nil
	}

	p.canvas.Edges = append(p.canvas.Edges, edge)
	return nil
}

func (p *CanvasPatcher) disconnectNodes(operation *pb.CanvasUpdateOperation) error {
	edge, err := edgeFromOperation(operation)
	if err != nil {
		return err
	}

	edgeIndex, found := p.findEdge(edge)
	if !found {
		return nil
	}

	p.canvas.Edges = slices.Delete(p.canvas.Edges, edgeIndex, edgeIndex+1)
	return nil
}

func (p *CanvasPatcher) findNode(nodeID string) (int, bool) {
	index := slices.IndexFunc(p.canvas.Nodes, func(node models.Node) bool {
		return node.ID == nodeID
	})

	return index, index >= 0
}

func (p *CanvasPatcher) findEdge(targetEdge models.Edge) (int, bool) {
	index := slices.IndexFunc(p.canvas.Edges, func(edge models.Edge) bool {
		return edge.SourceID == targetEdge.SourceID &&
			edge.TargetID == targetEdge.TargetID &&
			edge.Channel == targetEdge.Channel
	})

	return index, index >= 0
}

func (p *CanvasPatcher) findBlock(block string) (string, *models.NodeRef, error) {
	if block == "" {
		return "", nil, fmt.Errorf("block is required")
	}

	if _, err := uuid.Parse(block); err == nil {
		return models.NodeTypeBlueprint, &models.NodeRef{
			Blueprint: &models.BlueprintRef{ID: block},
		}, nil
	}

	if _, err := p.registry.GetComponent(block); err == nil {
		return models.NodeTypeComponent, &models.NodeRef{
			Component: &models.ComponentRef{Name: block},
		}, nil
	}

	if _, err := p.registry.GetTrigger(block); err == nil {
		return models.NodeTypeTrigger, &models.NodeRef{
			Trigger: &models.TriggerRef{Name: block},
		}, nil
	}

	if _, err := p.registry.GetWidget(block); err == nil {
		return models.NodeTypeWidget, &models.NodeRef{
			Widget: &models.WidgetRef{Name: block},
		}, nil
	}

	return "", nil, fmt.Errorf("block %s not found in registry", block)
}

func edgeFromOperation(operation *pb.CanvasUpdateOperation) (models.Edge, error) {
	source := operation.GetSource()
	target := operation.GetTarget()

	if source == nil || target == nil {
		return models.Edge{}, fmt.Errorf("source and target are required for %s", operation.Type)
	}

	sourceID := source.GetId()
	targetID := target.GetId()
	if sourceID == "" || targetID == "" {
		return models.Edge{}, fmt.Errorf("source and target node ids are required for %s", operation.Type)
	}

	sourceChannel := source.GetChannel()
	targetChannel := target.GetChannel()
	if sourceChannel != targetChannel {
		return models.Edge{}, fmt.Errorf("source and target channels must match for %s", operation.Type)
	}

	return models.Edge{
		SourceID: sourceID,
		TargetID: targetID,
		Channel:  sourceChannel,
	}, nil
}

func (p *CanvasPatcher) validateCanvasGraph() error {
	nodeIDs := make(map[string]bool, len(p.canvas.Nodes))
	inDegree := make(map[string]int, len(p.canvas.Nodes))
	adjacency := make(map[string][]string, len(p.canvas.Nodes))

	for _, node := range p.canvas.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node id is required")
		}

		if node.Name == "" {
			return fmt.Errorf("node %s name is required", node.ID)
		}

		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node id: %s", node.ID)
		}

		nodeIDs[node.ID] = true
		adjacency[node.ID] = []string{}
		inDegree[node.ID] = 0
	}

	for _, edge := range p.canvas.Edges {
		if edge.SourceID == "" || edge.TargetID == "" {
			return fmt.Errorf("source and target node ids are required")
		}

		if edge.SourceID == edge.TargetID {
			return fmt.Errorf("self-loop edges are not allowed")
		}

		if !nodeIDs[edge.SourceID] {
			return fmt.Errorf("source node %s not found", edge.SourceID)
		}

		if !nodeIDs[edge.TargetID] {
			return fmt.Errorf("target node %s not found", edge.TargetID)
		}

		adjacency[edge.SourceID] = append(adjacency[edge.SourceID], edge.TargetID)
		inDegree[edge.TargetID]++
	}

	queue := make([]string, 0, len(p.canvas.Nodes))
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	visitedCount := 0
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		visitedCount++

		for _, childNodeID := range adjacency[nodeID] {
			inDegree[childNodeID]--
			if inDegree[childNodeID] == 0 {
				queue = append(queue, childNodeID)
			}
		}
	}

	if visitedCount != len(p.canvas.Nodes) {
		return fmt.Errorf("graph contains a cycle")
	}

	return nil
}
