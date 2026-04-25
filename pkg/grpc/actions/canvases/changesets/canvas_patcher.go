package changesets

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type CanvasPatcher struct {
	tx              *gorm.DB
	orgID           uuid.UUID
	registry        *registry.Registry
	originalVersion *models.CanvasVersion
	finalVersion    *models.CanvasVersion

	//
	// Using maps to keep lookup operations fast
	//
	nodes map[string]models.Node
	edges map[string]models.Edge
}

func NewCanvasPatcher(tx *gorm.DB, orgID uuid.UUID, registry *registry.Registry, canvas *models.CanvasVersion) *CanvasPatcher {
	p := &CanvasPatcher{
		tx:              tx,
		orgID:           orgID,
		registry:        registry,
		originalVersion: canvas,
		nodes:           make(map[string]models.Node),
		edges:           make(map[string]models.Edge),
	}

	for _, node := range p.originalVersion.Nodes {
		p.nodes[node.ID] = node
	}

	for _, edge := range p.originalVersion.Edges {
		p.edges[p.edgeKey(edge.SourceID, edge.TargetID, edge.Channel)] = edge
	}

	return p
}

func (p *CanvasPatcher) edgeKey(sourceID, targetID, channel string) string {
	return sourceID + "|" + targetID + "|" + channel
}

func (p *CanvasPatcher) GetVersion() *models.CanvasVersion {
	return p.finalVersion
}

func (p *CanvasPatcher) buildFinalVersion(autoLayout *pb.CanvasAutoLayout) (*models.CanvasVersion, error) {
	v := &models.CanvasVersion{
		ID:          p.originalVersion.ID,
		WorkflowID:  p.originalVersion.WorkflowID,
		OwnerID:     p.originalVersion.OwnerID,
		State:       p.originalVersion.State,
		PublishedAt: p.originalVersion.PublishedAt,
		CreatedAt:   p.originalVersion.CreatedAt,
		UpdatedAt:   p.originalVersion.UpdatedAt,
	}

	nodeIDs := make([]string, 0, len(p.nodes))
	for nodeID := range p.nodes {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)

	v.Nodes = make([]models.Node, 0, len(p.nodes))
	for _, nodeID := range nodeIDs {
		v.Nodes = append(v.Nodes, p.nodes[nodeID])
	}

	edgeKeys := make([]string, 0, len(p.edges))
	for edgeKey := range p.edges {
		edgeKeys = append(edgeKeys, edgeKey)
	}
	sort.Strings(edgeKeys)

	v.Edges = make([]models.Edge, 0, len(p.edges))
	for _, edgeKey := range edgeKeys {
		v.Edges = append(v.Edges, p.edges[edgeKey])
	}

	if autoLayout == nil {
		return v, nil
	}

	nodes, edges, err := layout.ApplyLayout(v.Nodes, v.Edges, autoLayout)
	if err != nil {
		return nil, err
	}

	v.Nodes = nodes
	v.Edges = edges
	return v, nil
}

func (p *CanvasPatcher) ApplyChangeset(changeset *pb.CanvasChangeset, autoLayout *pb.CanvasAutoLayout) error {
	if changeset == nil || len(changeset.Changes) == 0 {
		return fmt.Errorf("changeset is required")
	}

	for _, change := range changeset.Changes {
		if err := p.handleChange(change); err != nil {
			return err
		}
	}

	finalVersion, err := p.buildFinalVersion(autoLayout)
	if err != nil {
		return err
	}

	p.finalVersion = finalVersion
	return CheckForCycles(p.finalVersion.Nodes, p.finalVersion.Edges)
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
	//
	// These initial checks are hard checks.
	// If they fail, we should return an error immediately.
	//
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

	if _, exists := p.nodes[nodeID]; exists {
		return fmt.Errorf("node %s already exists", nodeID)
	}

	newNode := models.Node{
		ID:          nodeID,
		Name:        node.GetName(),
		IsCollapsed: node.GetIsCollapsed(),
	}

	nodeType, nodeRef, err := p.findBlock(node)
	if err != nil {
		return fmt.Errorf("failed to find block: %v", err)
	}

	newNode.Type = nodeType
	newNode.Ref = *nodeRef

	if node.GetPosition() != nil {
		newNode.Position.X = int(node.GetPosition().GetX())
		newNode.Position.Y = int(node.GetPosition().GetY())
	}

	//
	// From here on out, we don't return errors,
	// we just save the error message in the node.ErrorMessage field.
	// This still allows the changeset to be applied, but
	// node will be in an error state.
	//

	integrationID, err := p.validateIntegration(node)
	if err != nil {
		errorMessage := err.Error()
		newNode.ErrorMessage = &errorMessage
		p.nodes[nodeID] = newNode
		return nil
	}

	newNode.IntegrationID = integrationID
	schema, err := p.findConfigurationSchemaForNode(nodeType, *nodeRef)
	if err != nil {
		errorMessage := err.Error()
		newNode.ErrorMessage = &errorMessage
		p.nodes[nodeID] = newNode
		return nil
	}

	var nodeConfiguration map[string]any
	if node.GetConfiguration() != nil {
		nodeConfiguration = node.GetConfiguration().AsMap()
	}

	err = configuration.ValidateConfiguration(schema, nodeConfiguration)
	if err != nil {
		errorMessage := err.Error()
		newNode.ErrorMessage = &errorMessage
		newNode.Configuration = nodeConfiguration
		p.nodes[nodeID] = newNode
		return nil
	}

	newNode.Configuration = nodeConfiguration
	p.nodes[nodeID] = newNode
	return nil
}

func (p *CanvasPatcher) validateIntegration(node *pb.CanvasChangeset_Change_Node) (*string, error) {
	if p.registry.IsCoreBlock(node.GetBlock()) {
		return nil, nil
	}

	if node.GetIntegrationId() == "" {
		return nil, fmt.Errorf("integration is required for %s", node.GetBlock())
	}

	integration := node.GetIntegrationId()
	integrationID, err := uuid.Parse(integration)
	if err != nil {
		return nil, fmt.Errorf("invalid integration id: %v", err)
	}

	_, err = models.FindIntegrationInTransaction(p.tx, p.orgID, integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration %s not found", node.GetIntegrationId())
	}

	return &integration, nil
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

	currentNode, exists := p.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	delete(p.nodes, nodeID)

	for edgeKey, edge := range p.edges {
		if edge.SourceID == currentNode.ID || edge.TargetID == currentNode.ID {
			delete(p.edges, edgeKey)
		}
	}

	return nil
}

func (p *CanvasPatcher) updateNode(change *pb.CanvasChangeset_Change) error {

	//
	// These initial checks are hard checks.
	// If they fail, we should return an error immediately.
	//
	node := change.GetNode()
	if node == nil {
		return fmt.Errorf("node is required for %s", change.Type)
	}

	nodeID := node.GetId()
	if nodeID == "" {
		return fmt.Errorf("node id is required for %s", change.Type)
	}

	currentNode, exists := p.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	if node.GetName() != "" {
		currentNode.Name = node.GetName()
	}

	if node.GetPosition() != nil {
		currentNode.Position.X = int(node.GetPosition().GetX())
		currentNode.Position.Y = int(node.GetPosition().GetY())
	}

	if node.IsCollapsed != nil {
		currentNode.IsCollapsed = node.GetIsCollapsed()
	}

	//
	// From here on out, we don't return errors,
	// we save the error message alongside the new invalid configuration.
	// This still allows the changeset to be applied, but
	// node will be in an error state.
	//

	if node.GetConfiguration() != nil {
		schema, err := p.findConfigurationSchemaForNode(currentNode.Type, currentNode.Ref)
		if err != nil {
			errorMessage := err.Error()
			currentNode.ErrorMessage = &errorMessage
			currentNode.Configuration = node.GetConfiguration().AsMap()
			p.nodes[nodeID] = currentNode
			return nil
		}

		err = configuration.ValidateConfiguration(schema, node.GetConfiguration().AsMap())
		if err != nil {
			errorMessage := err.Error()
			currentNode.ErrorMessage = &errorMessage
			currentNode.Configuration = node.GetConfiguration().AsMap()
			p.nodes[nodeID] = currentNode
			return nil
		}

		currentNode.Configuration = node.GetConfiguration().AsMap()
		currentNode.ErrorMessage = nil
	}

	p.nodes[nodeID] = currentNode
	return nil
}

func (p *CanvasPatcher) findConfigurationSchemaForNode(nodeType string, nodeRef models.NodeRef) ([]configuration.Field, error) {
	switch nodeType {
	case models.NodeTypeComponent:
		action, err := p.registry.GetAction(nodeRef.Component.Name)
		if err != nil {
			return nil, err
		}

		return action.Configuration(), nil

	case models.NodeTypeTrigger:
		trigger, err := p.registry.GetTrigger(nodeRef.Trigger.Name)
		if err != nil {
			return nil, err
		}

		return trigger.Configuration(), nil

	case models.NodeTypeWidget:
		widget, err := p.registry.GetWidget(nodeRef.Widget.Name)
		if err != nil {
			return nil, err
		}

		return widget.Configuration(), nil

	default:
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}
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

	if _, exists := p.nodes[edge.GetSourceId()]; !exists {
		return fmt.Errorf("source node %s not found", edge.GetSourceId())
	}

	if _, exists := p.nodes[edge.GetTargetId()]; !exists {
		return fmt.Errorf("target node %s not found", edge.GetTargetId())
	}

	if err := ValidateSourceNodeOutputChannel(
		p.registry,
		p.nodes[edge.GetSourceId()],
		edge.GetChannel(),
	); err != nil {
		return err
	}

	edgeKey := p.edgeKey(edge.GetSourceId(), edge.GetTargetId(), edge.GetChannel())
	if _, exists := p.edges[edgeKey]; exists {
		return nil
	}

	p.edges[edgeKey] = models.Edge{
		SourceID: edge.GetSourceId(),
		TargetID: edge.GetTargetId(),
		Channel:  edge.GetChannel(),
	}

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

	edgeKey := p.edgeKey(edge.GetSourceId(), edge.GetTargetId(), edge.GetChannel())
	if _, exists := p.edges[edgeKey]; !exists {
		return nil
	}

	delete(p.edges, edgeKey)
	return nil
}

func (p *CanvasPatcher) findBlock(node *pb.CanvasChangeset_Change_Node) (string, *models.NodeRef, error) {
	if node.GetBlock() == "" {
		return "", nil, fmt.Errorf("block is required")
	}

	//
	// Check if the block is an action
	//
	_, err := p.registry.GetAction(node.GetBlock())
	if err == nil {
		return models.NodeTypeComponent, &models.NodeRef{
			Component: &models.ComponentRef{Name: node.GetBlock()},
		}, nil
	}

	//
	// Otherwise, check if the block is a trigger
	//
	_, err = p.registry.GetTrigger(node.GetBlock())
	if err == nil {
		return models.NodeTypeTrigger, &models.NodeRef{
			Trigger: &models.TriggerRef{Name: node.GetBlock()},
		}, nil
	}

	//
	// Otherwise, check if the block is a widget
	//
	_, err = p.registry.GetWidget(node.GetBlock())
	if err == nil {
		return models.NodeTypeWidget, &models.NodeRef{
			Widget: &models.WidgetRef{Name: node.GetBlock()},
		}, nil
	}

	//
	// If the block is not any of the above, return an error
	//
	return "", nil, fmt.Errorf("block %s not found in registry", node.GetBlock())
}
