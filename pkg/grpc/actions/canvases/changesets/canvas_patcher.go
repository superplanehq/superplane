package changesets

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

type CanvasPatcherOptions struct {
	OrgID             uuid.UUID
	Registry          *registry.Registry
	Encryptor         crypto.Encryptor
	BaseURL           string
	AuthService       authorization.Authorization
	AuthenticatedUser *models.User
}

func (o *CanvasPatcherOptions) Validate() error {
	if o.OrgID == uuid.Nil {
		return fmt.Errorf("org ID is required")
	}

	if o.Encryptor == nil {
		return fmt.Errorf("encryptor is required")
	}

	if o.Registry == nil {
		return fmt.Errorf("registry is required")
	}

	if o.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	if o.AuthService == nil {
		return fmt.Errorf("auth service is required")
	}

	if o.AuthenticatedUser == nil {
		return fmt.Errorf("authenticated user is required")
	}

	return nil
}

type CanvasPatcher struct {
	tx              *gorm.DB
	options         *CanvasPatcherOptions
	originalVersion *models.CanvasVersion
	finalVersion    *models.CanvasVersion

	//
	// Using maps to keep lookup operations fast
	//
	nodes map[string]models.Node
	edges map[string]models.Edge
}

func NewCanvasPatcher(tx *gorm.DB, canvas *models.CanvasVersion, options *CanvasPatcherOptions) (*CanvasPatcher, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	p := &CanvasPatcher{
		tx:              tx,
		options:         options,
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

	return p, nil
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

	integration, err := p.validateIntegration(node)
	if err != nil {
		errorMessage := err.Error()
		newNode.ErrorMessage = &errorMessage
		p.nodes[nodeID] = newNode
		return nil
	}

	integrationID := integration.ID.String()
	newNode.IntegrationID = &integrationID
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

	newNode.Configuration = nodeConfiguration
	err = configuration.ValidateConfiguration(schema, nodeConfiguration)
	if err != nil {
		errorMessage := err.Error()
		newNode.ErrorMessage = &errorMessage
		p.nodes[nodeID] = newNode
		return nil
	}

	//
	// Run Setup() for the node
	//
	err = p.setupNode(integration, newNode)
	if err != nil {
		errorMessage := err.Error()
		newNode.ErrorMessage = &errorMessage
	}

	p.nodes[nodeID] = newNode
	return nil
}

func (p *CanvasPatcher) validateIntegration(node *pb.CanvasChangeset_Change_Node) (*models.Integration, error) {
	if p.options.Registry.IsCoreBlock(node.GetBlock()) {
		return nil, nil
	}

	if node.GetIntegrationId() == "" {
		return nil, fmt.Errorf("integration is required for %s", node.GetBlock())
	}

	integrationID, err := uuid.Parse(node.GetIntegrationId())
	if err != nil {
		return nil, fmt.Errorf("invalid integration id: %v", err)
	}

	integration, err := models.FindIntegrationInTransaction(p.tx, p.options.OrgID, integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration %s not found", node.GetIntegrationId())
	}

	return integration, nil
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

	integration, err := p.validateIntegration(node)
	if err != nil {
		errorMessage := err.Error()
		currentNode.ErrorMessage = &errorMessage
		p.nodes[nodeID] = currentNode
		return nil
	}

	integrationID := integration.ID.String()
	currentNode.IntegrationID = &integrationID

	if node.GetConfiguration() == nil {
		p.nodes[nodeID] = currentNode
		return nil
	}

	currentNode.Configuration = node.GetConfiguration().AsMap()
	schema, err := p.findConfigurationSchemaForNode(currentNode.Type, currentNode.Ref)
	if err != nil {
		errorMessage := err.Error()
		currentNode.ErrorMessage = &errorMessage
		p.nodes[nodeID] = currentNode
		return nil
	}

	err = configuration.ValidateConfiguration(schema, node.GetConfiguration().AsMap())
	if err != nil {
		errorMessage := err.Error()
		currentNode.ErrorMessage = &errorMessage
		p.nodes[nodeID] = currentNode
		return nil
	}

	err = p.setupNode(integration, currentNode)
	if err != nil {
		errorMessage := err.Error()
		currentNode.ErrorMessage = &errorMessage
		p.nodes[nodeID] = currentNode
		return nil
	}

	currentNode.ErrorMessage = nil
	p.nodes[nodeID] = currentNode
	return nil
}

func (p *CanvasPatcher) findConfigurationSchemaForNode(nodeType string, nodeRef models.NodeRef) ([]configuration.Field, error) {
	switch nodeType {
	case models.NodeTypeComponent:
		component, err := p.options.Registry.GetComponent(nodeRef.Component.Name)
		if err != nil {
			return nil, err
		}

		return component.Configuration(), nil

	case models.NodeTypeTrigger:
		trigger, err := p.options.Registry.GetTrigger(nodeRef.Trigger.Name)
		if err != nil {
			return nil, err
		}

		return trigger.Configuration(), nil

	case models.NodeTypeWidget:
		widget, err := p.options.Registry.GetWidget(nodeRef.Widget.Name)
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
	// Check if the block is a component
	//
	_, err := p.options.Registry.GetComponent(node.GetBlock())
	if err == nil {
		return models.NodeTypeComponent, &models.NodeRef{
			Component: &models.ComponentRef{Name: node.GetBlock()},
		}, nil
	}

	//
	// Otherwise, check if the block is a trigger
	//
	_, err = p.options.Registry.GetTrigger(node.GetBlock())
	if err == nil {
		return models.NodeTypeTrigger, &models.NodeRef{
			Trigger: &models.TriggerRef{Name: node.GetBlock()},
		}, nil
	}

	//
	// Otherwise, check if the block is a widget
	//
	_, err = p.options.Registry.GetWidget(node.GetBlock())
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

func (p *CanvasPatcher) setupNode(integration *models.Integration, node models.Node) error {
	switch node.Type {
	case models.NodeTypeComponent:
		return p.setupComponent(integration, node)
	case models.NodeTypeTrigger:
		return p.setupTrigger(integration, node)
	case models.NodeTypeWidget:
		return nil
	}

	return fmt.Errorf("unknown node type: %s", node.Type)
}

func (p *CanvasPatcher) setupComponent(integration *models.Integration, node models.Node) error {
	component, err := p.options.Registry.GetComponent(node.Ref.Component.Name)
	if err != nil {
		return err
	}

	setupCtx := core.SetupContext{
		Logger:        &logrus.Entry{},
		Configuration: node.Configuration,
		HTTP:          p.options.Registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataReader(node.Metadata),
		Requests:      contexts.NewNoOpRequestContext(),
		Webhook:       contexts.NewNoOpNodeWebhookContext(p.options.BaseURL),
		Auth:          contexts.NewAuthReader(p.tx, p.options.OrgID, p.options.AuthService, p.options.AuthenticatedUser),
		Integration:   contexts.NewIntegrationReader(p.tx, nil, integration, p.options.Encryptor, p.options.Registry),
	}

	return component.Setup(setupCtx)
}

func (p *CanvasPatcher) setupTrigger(integration *models.Integration, node models.Node) error {
	component, err := p.options.Registry.GetTrigger(node.Ref.Trigger.Name)
	if err != nil {
		return err
	}

	setupCtx := core.TriggerContext{
		Logger:        &logrus.Entry{},
		Configuration: node.Configuration,
		HTTP:          p.options.Registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataReader(node.Metadata),
		Requests:      contexts.NewNoOpRequestContext(),
		Webhook:       contexts.NewNoOpNodeWebhookContext(p.options.BaseURL),
		Integration:   contexts.NewIntegrationReader(p.tx, nil, integration, p.options.Encryptor, p.options.Registry),
		Events:        contexts.NewNoOpEventContext(),
	}

	return component.Setup(setupCtx)
}
