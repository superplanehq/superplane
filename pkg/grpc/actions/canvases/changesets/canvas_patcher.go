package changesets

import (
	"fmt"
	"slices"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
)

type CanvasPatcher struct {
	registry *registry.Registry
	canvas   *models.CanvasVersion
}

func NewCanvasPatcher(registry *registry.Registry, canvas *models.CanvasVersion) *CanvasPatcher {
	return &CanvasPatcher{
		registry: registry,
		canvas:   canvas,
	}
}

func (p *CanvasPatcher) GetVersion() *models.CanvasVersion {
	return p.canvas
}

func (p *CanvasPatcher) ApplyChangeset(changeset *pb.CanvasChangeset) error {
	if changeset == nil || len(changeset.Changes) == 0 {
		return nil
	}

	for _, change := range changeset.Changes {
		err := p.handleChange(change)
		if err != nil {
			return err
		}
	}

	return p.validateCanvasGraph()
}

func (p *CanvasPatcher) handleChange(change *pb.CanvasChangeset_Change) error {
	if change == nil {
		return fmt.Errorf("change is required")
	}

	switch change.Type {
	case pb.CanvasChangeset_Change_ADD_NODE:
		return p.addNode(change)
	case pb.CanvasChangeset_Change_DELETE_NODE:
		return p.deleteNode(change)
	case pb.CanvasChangeset_Change_UPDATE_NODE:
		return p.updateNode(change)
	case pb.CanvasChangeset_Change_ADD_EDGE:
		return p.addEdge(change)
	case pb.CanvasChangeset_Change_DELETE_EDGE:
		return p.deleteEdge(change)
	}

	return fmt.Errorf("unknown change type: %s", change.Type)
}

func (p *CanvasPatcher) addNode(change *pb.CanvasChangeset_Change) error {
	node := change.GetNode()
	if node == nil {
		return fmt.Errorf("node is required for %s", change.Type)
	}

	nodeID := node.GetId()
	if nodeID == "" {
		return fmt.Errorf("target node id is required for %s", change.Type)
	}

	if node.GetName() == "" {
		return fmt.Errorf("target node name is required for %s", change.Type)
	}

	if _, found := p.findNode(nodeID); found {
		return fmt.Errorf("node %s already exists", nodeID)
	}

	nodeType, nodeRef, err := p.findAndValidateBlock(node)
	if err != nil {
		return err
	}

	var configuration map[string]any
	if node.GetConfiguration() != nil {
		configuration = node.GetConfiguration().AsMap()
	}

	p.canvas.Nodes = append(p.canvas.Nodes, models.Node{
		ID:            nodeID,
		Name:          node.GetName(),
		Type:          nodeType,
		Ref:           *nodeRef,
		Configuration: configuration,
	})

	return nil
}

func (p *CanvasPatcher) deleteNode(change *pb.CanvasChangeset_Change) error {
	node := change.GetNode()
	if node == nil {
		return fmt.Errorf("target is required for %s", change.Type)
	}

	nodeID := node.GetId()
	if nodeID == "" {
		return fmt.Errorf("target node id is required for %s", change.Type)
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

func (p *CanvasPatcher) updateNode(change *pb.CanvasChangeset_Change) error {
	node := change.GetNode()
	if node == nil {
		return fmt.Errorf("node is required for %s", change.Type)
	}

	nodeID := node.GetId()
	if nodeID == "" {
		return fmt.Errorf("node id is required for %s", change.Type)
	}

	if node.GetName() == "" {
		return fmt.Errorf("node name is required for %s", change.Type)
	}

	nodeIndex, found := p.findNode(nodeID)
	if !found {
		return fmt.Errorf("node %s not found", nodeID)
	}

	p.canvas.Nodes[nodeIndex].Name = node.GetName()

	//
	// We only update the configuration if it is provided
	//
	if node.GetConfiguration() != nil {
		p.canvas.Nodes[nodeIndex].Configuration = node.GetConfiguration().AsMap()
	}

	return nil
}

func (p *CanvasPatcher) addEdge(change *pb.CanvasChangeset_Change) error {
	edge := change.GetEdge()
	if edge == nil {
		return fmt.Errorf("edge is required for %s", change.Type)
	}

	if edge.GetSourceId() == "" {
		return fmt.Errorf("source id is required for %s", change.Type)
	}

	if edge.GetTargetId() == "" {
		return fmt.Errorf("target id is required for %s", change.Type)
	}

	if edge.GetChannel() == "" {
		return fmt.Errorf("channel is required for %s", change.Type)
	}

	if edge.GetSourceId() == edge.GetTargetId() {
		return fmt.Errorf("self-loop edges are not allowed")
	}

	if _, found := p.findNode(edge.GetSourceId()); !found {
		return fmt.Errorf("source node %s not found", edge.GetSourceId())
	}

	if _, found := p.findNode(edge.GetTargetId()); !found {
		return fmt.Errorf("target node %s not found", edge.GetTargetId())
	}

	if _, found := p.findEdge(edge.GetSourceId(), edge.GetTargetId(), edge.GetChannel()); found {
		return nil
	}

	p.canvas.Edges = append(p.canvas.Edges, models.Edge{
		SourceID: edge.GetSourceId(),
		TargetID: edge.GetTargetId(),
		Channel:  edge.GetChannel(),
	})

	return nil
}

func (p *CanvasPatcher) deleteEdge(change *pb.CanvasChangeset_Change) error {
	edge := change.GetEdge()
	if edge == nil {
		return fmt.Errorf("edge is required for %s", change.Type)
	}

	if edge.GetSourceId() == "" {
		return fmt.Errorf("source id is required for %s", change.Type)
	}

	if edge.GetTargetId() == "" {
		return fmt.Errorf("target id is required for %s", change.Type)
	}

	if edge.GetChannel() == "" {
		return fmt.Errorf("channel is required for %s", change.Type)
	}

	edgeIndex, found := p.findEdge(edge.GetSourceId(), edge.GetTargetId(), edge.GetChannel())
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

func (p *CanvasPatcher) findEdge(sourceID, targetID, channel string) (int, bool) {
	index := slices.IndexFunc(p.canvas.Edges, func(edge models.Edge) bool {
		return edge.SourceID == sourceID &&
			edge.TargetID == targetID &&
			edge.Channel == channel
	})

	return index, index >= 0
}

func (p *CanvasPatcher) findAndValidateBlock(node *pb.CanvasChangeset_Change_Node) (string, *models.NodeRef, error) {
	if node.GetBlock() == "" {
		return "", nil, fmt.Errorf("block is required")
	}

	//
	// Check if the block is a component
	//
	component, err := p.registry.GetComponent(node.GetBlock())
	if err == nil {
		err = configuration.ValidateConfiguration(component.Configuration(), node.GetConfiguration().AsMap())
		if err != nil {
			return "", nil, err
		}

		return models.NodeTypeComponent, &models.NodeRef{
			Component: &models.ComponentRef{Name: node.GetBlock()},
		}, nil
	}

	//
	// Otherwise, check if the block is a trigger
	//
	trigger, err := p.registry.GetTrigger(node.GetBlock())
	if err == nil {
		err = configuration.ValidateConfiguration(trigger.Configuration(), node.GetConfiguration().AsMap())
		if err != nil {
			return "", nil, err
		}

		return models.NodeTypeTrigger, &models.NodeRef{
			Trigger: &models.TriggerRef{Name: node.GetBlock()},
		}, nil
	}

	//
	// Otherwise, check if the block is a widget
	//
	widget, err := p.registry.GetWidget(node.GetBlock())
	if err == nil {
		err = configuration.ValidateConfiguration(widget.Configuration(), node.GetConfiguration().AsMap())
		if err != nil {
			return "", nil, err
		}

		return models.NodeTypeWidget, &models.NodeRef{
			Widget: &models.WidgetRef{Name: node.GetBlock()},
		}, nil
	}

	//
	// If the block is not any of the above, return an error
	//
	return "", nil, fmt.Errorf("block %s not found in registry", node.GetBlock())
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
		return fmt.Errorf("canvas contains a cycle")
	}

	return nil
}
