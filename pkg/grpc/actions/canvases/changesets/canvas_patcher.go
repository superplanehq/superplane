package changesets

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
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

func (c *CanvasPatcherOptions) Validate() error {
	if c.OrgID == uuid.Nil {
		return fmt.Errorf("organization id is required")
	}

	if c.Registry == nil {
		return fmt.Errorf("registry is required")
	}

	if c.Encryptor == nil {
		return fmt.Errorf("encryptor is required")
	}

	if c.BaseURL == "" {
		return fmt.Errorf("baseURL is required")
	}

	return nil
}

type CanvasPatcher struct {
	config          *CanvasPatcherOptions
	originalVersion *models.CanvasVersion
	finalVersion    *models.CanvasVersion

	//
	// Using maps to keep lookup operations fast
	//
	nodes map[string]models.Node
	edges map[string]models.Edge
}

func NewCanvasPatcher(config *CanvasPatcherOptions, canvas *models.CanvasVersion) (*CanvasPatcher, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	err := config.Validate()
	if err != nil {
		return nil, err
	}

	p := &CanvasPatcher{
		config:          config,
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

	nodes, edges, err := layout.ApplyLayout(v.Nodes, v.Edges, autoLayout)
	if err != nil {
		return nil, err
	}

	v.Nodes = nodes
	v.Edges = edges
	return v, nil
}

func (p *CanvasPatcher) ApplyChangeset(tx *gorm.DB, changeset *pb.CanvasChangeset, autoLayout *pb.CanvasAutoLayout) error {
	if changeset == nil || len(changeset.Changes) == 0 {
		return fmt.Errorf("changeset is required")
	}

	for _, change := range changeset.Changes {
		if err := p.handleChange(tx, change); err != nil {
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

func (p *CanvasPatcher) handleChange(tx *gorm.DB, change *pb.CanvasChangeset_Change) error {
	if change == nil {
		return fmt.Errorf("change is required")
	}

	switch change.Type {
	case pb.CanvasChangeset_Change_ADD_NODE:
		return p.addNode(tx, change)
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

func (p *CanvasPatcher) addNode(tx *gorm.DB, change *pb.CanvasChangeset_Change) error {
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
		ID:   nodeID,
		Name: node.GetName(),
	}

	nodeType, nodeRef, err := p.findBlock(node)
	if err != nil {
		return err
	}

	newNode.Type = nodeType
	newNode.Ref = *nodeRef

	integration, err := p.validateIntegration(tx, node)
	if err != nil {
		return err
	}

	if integration != nil {
		integrationID := integration.ID.String()
		newNode.IntegrationID = &integrationID
	}

	schema, err := p.findConfigurationSchemaForNode(nodeType, *nodeRef)
	if err != nil {
		return err
	}

	var nodeConfiguration map[string]any
	if node.GetConfiguration() != nil {
		nodeConfiguration = node.GetConfiguration().AsMap()
	}

	err = configuration.ValidateConfiguration(schema, nodeConfiguration)
	if err != nil {
		return err
	}

	newNode.Configuration = nodeConfiguration

	err = p.setupNode(tx, &newNode, integration)
	if err != nil {
		return err
	}

	p.nodes[nodeID] = newNode

	return nil
}

func (p *CanvasPatcher) setupNode(tx *gorm.DB, node *models.Node, integration *models.Integration) error {
	switch node.Type {
	case models.NodeTypeComponent:
		return p.setupComponent(tx, node, integration)
	case models.NodeTypeTrigger:
		return p.setupTrigger(tx, node, integration)
	}

	return nil
}

func (p *CanvasPatcher) setupComponent(tx *gorm.DB, node *models.Node, integration *models.Integration) error {
	component, err := p.config.Registry.GetComponent(node.Ref.Component.Name)
	if err != nil {
		return err
	}

	return component.Setup(core.SetupContext{
		Configuration: node.Configuration,
		HTTP:          p.config.Registry.HTTPContext(),
		Metadata:      contexts.NewReadOnlyNodeMetadataContext(node.Metadata),
		Requests:      contexts.NewNoOpNodeRequestContext(),
		Webhook:       contexts.NewNoOpNodeWebhookContext(p.config.BaseURL),
		Integration:   contexts.NewReadOnlyIntegrationContext(tx, p.config.Encryptor, p.config.Registry, integration),

		//
		// NOTE: auth is a read-only context already, so we can use it directly.
		//
		Auth: contexts.NewAuthContext(tx, p.config.OrgID, p.config.AuthService, p.config.AuthenticatedUser),
	})
}

func (p *CanvasPatcher) setupTrigger(tx *gorm.DB, node *models.Node, integration *models.Integration) error {
	trigger, err := p.config.Registry.GetTrigger(node.Ref.Trigger.Name)
	if err != nil {
		return err
	}

	return trigger.Setup(core.TriggerContext{
		Configuration: node.Configuration,
		HTTP:          p.config.Registry.HTTPContext(),
		Metadata:      contexts.NewReadOnlyNodeMetadataContext(node.Metadata),
		Requests:      contexts.NewNoOpNodeRequestContext(),
		Webhook:       contexts.NewNoOpNodeWebhookContext(p.config.BaseURL),
		Integration:   contexts.NewReadOnlyIntegrationContext(tx, p.config.Encryptor, p.config.Registry, integration),
		Events:        contexts.NewNoOpEventContext(),
	})
}

func (p *CanvasPatcher) validateIntegration(tx *gorm.DB, node *pb.CanvasChangeset_Change_Node) (*models.Integration, error) {
	//
	// Core blocks do not require an integration
	//
	if p.config.Registry.IsCoreBlock(node.GetBlock()) {
		return nil, nil
	}

	//
	// Otherwise, integration is required
	//
	if node.GetIntegrationId() == "" {
		return nil, fmt.Errorf("integration is required for %s", node.GetBlock())
	}

	integrationID, err := uuid.Parse(node.GetIntegrationId())
	if err != nil {
		return nil, fmt.Errorf("invalid integration id: %v", err)
	}

	integration, err := models.FindIntegrationInTransaction(tx, p.config.OrgID, integrationID)
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

	currentNode, exists := p.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	currentNode.Name = node.GetName()

	//
	// We only update the configuration if it is provided
	//
	if node.GetConfiguration() != nil {
		schema, err := p.findConfigurationSchemaForNode(currentNode.Type, currentNode.Ref)
		if err != nil {
			return err
		}

		err = configuration.ValidateConfiguration(schema, node.GetConfiguration().AsMap())
		if err != nil {
			return err
		}

		currentNode.Configuration = node.GetConfiguration().AsMap()
	}

	p.nodes[nodeID] = currentNode
	return nil
}

func (p *CanvasPatcher) findConfigurationSchemaForNode(nodeType string, nodeRef models.NodeRef) ([]configuration.Field, error) {
	switch nodeType {
	case models.NodeTypeComponent:
		component, err := p.config.Registry.GetComponent(nodeRef.Component.Name)
		if err != nil {
			return nil, err
		}

		return component.Configuration(), nil

	case models.NodeTypeTrigger:
		trigger, err := p.config.Registry.GetTrigger(nodeRef.Trigger.Name)
		if err != nil {
			return nil, err
		}

		return trigger.Configuration(), nil

	case models.NodeTypeWidget:
		widget, err := p.config.Registry.GetWidget(nodeRef.Widget.Name)
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
	_, err := p.config.Registry.GetComponent(node.GetBlock())
	if err == nil {
		return models.NodeTypeComponent, &models.NodeRef{
			Component: &models.ComponentRef{Name: node.GetBlock()},
		}, nil
	}

	//
	// Otherwise, check if the block is a trigger
	//
	_, err = p.config.Registry.GetTrigger(node.GetBlock())
	if err == nil {
		return models.NodeTypeTrigger, &models.NodeRef{
			Trigger: &models.TriggerRef{Name: node.GetBlock()},
		}, nil
	}

	//
	// Otherwise, check if the block is a widget
	//
	_, err = p.config.Registry.GetWidget(node.GetBlock())
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
