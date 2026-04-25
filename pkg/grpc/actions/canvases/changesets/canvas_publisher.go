package changesets

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/*
 * CanvasPublisher takes the live version and the proposed version,
 * calculates the changeset to go from the live version to the proposed version,
 * and applies any database changes required for that to happen.
 *
 * The workflow_versions.nodes already contains the state of the nodes,
 * so we don't need to run the same checks the CanvasPatcher does.
 *
 * Here, we only take the current state in the workflow_versions.nodes column,
 * run Setup(), and update things accordingly.
 */
type CanvasPublisher struct {
	tx         *gorm.DB
	options    CanvasPublisherOptions
	live       *models.CanvasVersion
	draft      *models.CanvasVersion
	changeset  *pb.CanvasChangeset
	finalNodes map[string]models.Node

	//
	// All nodes in the workflow, including deleted ones.
	// Deleted ones are needed, so we can use proper node IDs
	// when adding new nodes, ensuring they are unique and do
	// not conflict with deleted nodes.
	//
	allNodes map[string]models.CanvasNode
}

type CanvasPublisherOptions struct {
	Registry       *registry.Registry
	OrgID          uuid.UUID
	Encryptor      crypto.Encryptor
	AuthService    authorization.Authorization
	WebhookBaseURL string
}

func (o *CanvasPublisherOptions) Validate() error {
	if o.Registry == nil {
		return fmt.Errorf("registry is required")
	}

	if o.OrgID == uuid.Nil {
		return fmt.Errorf("org ID is required")
	}

	if o.Encryptor == nil {
		return fmt.Errorf("encryptor is required")
	}

	if o.AuthService == nil {
		return fmt.Errorf("auth service is required")
	}

	if o.WebhookBaseURL == "" {
		return fmt.Errorf("webhook base URL is required")
	}

	return nil
}

func NewCanvasPublisher(tx *gorm.DB, draft *models.CanvasVersion, liveVersion *models.CanvasVersion, options CanvasPublisherOptions) (*CanvasPublisher, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	changeset, err := NewChangesetBuilder(liveVersion.Nodes, liveVersion.Edges, draft.Nodes, draft.Edges).Build()
	if err != nil {
		return nil, err
	}

	if changeset == nil || len(changeset.Changes) == 0 {
		return nil, fmt.Errorf("no changes between live and draft version being applied")
	}

	allNodes, err := models.FindCanvasNodesUnscopedInTransaction(tx, liveVersion.WorkflowID)
	if err != nil {
		return nil, err
	}

	allNodesMap := make(map[string]models.CanvasNode)
	for _, node := range allNodes {
		allNodesMap[node.NodeID] = node
	}

	//
	// Since Setup() is only run here, we might need to update the
	// state of the []models.Node objects before saving them into
	// the workflow_versions.nodes column.
	//
	finalNodes := make(map[string]models.Node, len(draft.Nodes))
	for _, node := range draft.Nodes {
		finalNodes[node.ID] = node
	}

	return &CanvasPublisher{
		tx:         tx,
		options:    options,
		live:       liveVersion,
		finalNodes: finalNodes,
		draft:      draft,
		changeset:  changeset,
		allNodes:   allNodesMap,
	}, nil
}

func (p *CanvasPublisher) Publish(ctx context.Context) error {
	for _, change := range p.changeset.Changes {
		if err := p.processChange(ctx, change); err != nil {
			return err
		}
	}

	finalNodes := make([]models.Node, 0, len(p.finalNodes))
	for _, node := range p.finalNodes {
		finalNodes = append(finalNodes, node)
	}

	finalEdges := p.filterEdgesForExistingNodes(p.draft.Edges)
	err := models.PromoteToLiveInTransaction(p.tx, p.draft, finalNodes, finalEdges)
	if err != nil {
		return err
	}

	return nil
}

func (p *CanvasPublisher) filterEdgesForExistingNodes(edges []models.Edge) []models.Edge {
	filteredEdges := make([]models.Edge, 0, len(edges))
	for _, edge := range edges {
		if _, sourceExists := p.finalNodes[edge.SourceID]; !sourceExists {
			continue
		}

		if _, targetExists := p.finalNodes[edge.TargetID]; !targetExists {
			continue
		}

		filteredEdges = append(filteredEdges, edge)
	}

	return filteredEdges
}

func (p *CanvasPublisher) processChange(ctx context.Context, change *pb.CanvasChangeset_Change) error {
	switch change.Type {
	case pb.CanvasChangeset_Change_ADD_NODE:
		return p.addNode(ctx, change)
	case pb.CanvasChangeset_Change_DELETE_NODE:
		return p.deleteNode(change)
	case pb.CanvasChangeset_Change_UPDATE_NODE:
		return p.updateNode(ctx, change)
	}

	return nil
}

func (p *CanvasPublisher) addNode(ctx context.Context, change *pb.CanvasChangeset_Change) error {
	node := p.finalNodes[change.GetNode().GetId()]
	nodeID := p.ensureNewNodeID(node)
	node.ID = nodeID

	//
	// Widget nodes are not saved as workflow_nodes records in the database.
	//
	if node.Type == models.NodeTypeWidget {
		return nil
	}

	//
	// TODO: handle blueprint nodes once blueprints are enabled again
	//

	now := time.Now()
	newNode := models.CanvasNode{
		WorkflowID:        p.live.WorkflowID,
		NodeID:            nodeID,
		Name:              node.Name,
		Type:              node.Type,
		Ref:               datatypes.NewJSONType(node.Ref),
		Configuration:     datatypes.NewJSONType(node.Configuration),
		Metadata:          datatypes.NewJSONType(node.Metadata),
		Position:          datatypes.NewJSONType(node.Position),
		IsCollapsed:       node.IsCollapsed,
		AppInstallationID: p.getNodeIntegrationID(node),
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}

	//
	// If node update led to an error, set the node to error state.
	// Otherwise, node is ready.
	//
	if node.ErrorMessage != nil && strings.TrimSpace(*node.ErrorMessage) != "" {
		newNode.State = models.CanvasNodeStateError
		newNode.StateReason = node.ErrorMessage
	} else {
		newNode.State = models.CanvasNodeStateReady
		newNode.StateReason = nil
	}

	//
	// When adding a node, we need to insert it first,
	// so when we Setup() it, the contexts can create
	// records pointing to the new workflow_node record.
	//
	err := p.tx.Create(&newNode).Error
	if err != nil {
		return err
	}

	//
	// If node is already in error state, no need to run Setup() for it.
	//
	if newNode.State == models.CanvasNodeStateError {
		node.Metadata = newNode.Metadata.Data()
		p.finalNodes[node.ID] = node
		return nil
	}

	//
	// Otherwise, run Setup() for the node.
	//
	// If an error happens when setting up the node, we propagate that into
	// the finalNodes that are going to be saved into workflow_versions.nodes.
	//
	err = p.setupNode(ctx, &newNode)
	if err != nil {
		errorMsg := err.Error()
		newNode.State = models.CanvasNodeStateError
		newNode.StateReason = &errorMsg
		node.ErrorMessage = &errorMsg
	}

	node.Metadata = newNode.Metadata.Data()
	p.finalNodes[node.ID] = node
	return p.tx.Save(&newNode).Error
}

func (p *CanvasPublisher) updateNode(ctx context.Context, change *pb.CanvasChangeset_Change) error {
	updatedNode := p.finalNodes[change.GetNode().GetId()]

	//
	// Widgets are not saved as workflow_nodes records in the database.
	//
	if updatedNode.Type == models.NodeTypeWidget {
		return nil
	}

	existingNode, exists := p.allNodes[updatedNode.ID]
	if !exists {
		return fmt.Errorf("node %s not found", updatedNode.ID)
	}

	//
	// TODO: handle blueprint nodes once blueprints are enabled again
	//

	//
	// If node update led to an error, set the node to error state.
	// Otherwise, clear the error state.
	//
	if updatedNode.ErrorMessage != nil && strings.TrimSpace(*updatedNode.ErrorMessage) != "" {
		existingNode.State = models.CanvasNodeStateError
		existingNode.StateReason = updatedNode.ErrorMessage
	} else if existingNode.State == models.CanvasNodeStateError {
		existingNode.State = models.CanvasNodeStateReady
		existingNode.StateReason = nil
	}

	//
	// Update the node with the new values.
	//
	now := time.Now()
	existingNode.Name = updatedNode.Name
	existingNode.Type = updatedNode.Type
	existingNode.Ref = datatypes.NewJSONType(updatedNode.Ref)
	existingNode.Configuration = datatypes.NewJSONType(updatedNode.Configuration)
	existingNode.Position = datatypes.NewJSONType(updatedNode.Position)
	existingNode.IsCollapsed = updatedNode.IsCollapsed
	existingNode.AppInstallationID = p.getNodeIntegrationID(updatedNode)
	existingNode.UpdatedAt = &now

	//
	// If node is already in error state, no need to run Setup() for it.
	//
	if existingNode.State == models.CanvasNodeStateError {
		updatedNode.Metadata = existingNode.Metadata.Data()
		p.finalNodes[existingNode.NodeID] = updatedNode
		return p.tx.Save(&existingNode).Error
	}

	//
	// If an error happens when setting up the node,
	// we propagate that into the finalNodes that are
	// going to be saved into workflow_versions.nodes.
	//
	err := p.setupNode(ctx, &existingNode)
	if err != nil {
		errorMsg := err.Error()
		existingNode.State = models.CanvasNodeStateError
		existingNode.StateReason = &errorMsg
		updatedNode.ErrorMessage = &errorMsg
	}

	updatedNode.Metadata = existingNode.Metadata.Data()
	p.finalNodes[existingNode.NodeID] = updatedNode
	return p.tx.Save(&existingNode).Error
}

func (p *CanvasPublisher) deleteNode(change *pb.CanvasChangeset_Change) error {
	existingNode, exists := p.allNodes[change.GetNode().GetId()]
	if !exists {
		return nil
	}

	delete(p.allNodes, existingNode.NodeID)
	return models.DeleteCanvasNode(p.tx, existingNode)
}

func (p *CanvasPublisher) getNodeIntegrationID(node models.Node) *uuid.UUID {
	//
	// Only integration-based nodes have an integration ID,
	// so we must return nil for other node types.
	//
	if node.IntegrationID == nil || strings.TrimSpace(*node.IntegrationID) == "" {
		return nil
	}

	id := uuid.MustParse(*node.IntegrationID)
	return &id
}

func (p *CanvasPublisher) setupNode(ctx context.Context, node *models.CanvasNode) error {
	switch node.Type {
	case models.NodeTypeTrigger:
		return p.setupTrigger(ctx, node)
	case models.NodeTypeComponent:
		return p.setupAction(ctx, node)
	case models.NodeTypeWidget:
		return nil
	}

	return nil
}

func (p *CanvasPublisher) setupTrigger(ctx context.Context, node *models.CanvasNode) error {
	ref := node.Ref.Data()
	trigger, err := p.options.Registry.GetTrigger(ref.Trigger.Name)
	if err != nil {
		return err
	}

	logger := logging.ForNode(*node)
	triggerCtx := core.TriggerContext{
		Configuration: node.Configuration.Data(),
		HTTP:          p.options.Registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataContext(p.tx, node),
		Requests:      contexts.NewNodeRequestContext(p.tx, node),
		Events:        contexts.NewEventContext(p.tx, node, nil),
		Webhook:       contexts.NewNodeWebhookContext(ctx, p.tx, p.options.Encryptor, node, p.options.WebhookBaseURL),
	}

	if node.AppInstallationID != nil {
		integration, err := models.FindUnscopedIntegrationInTransaction(p.tx, *node.AppInstallationID)
		if err != nil {
			return fmt.Errorf("failed to find app installation: %v", err)
		}

		logger = logging.WithIntegration(logger, *integration)
		triggerCtx.Integration = contexts.NewIntegrationContext(
			p.tx,
			node,
			integration,
			p.options.Encryptor,
			p.options.Registry,
			nil,
		)
	}

	triggerCtx.Logger = logger
	return trigger.Setup(triggerCtx)
}

func (p *CanvasPublisher) setupAction(ctx context.Context, node *models.CanvasNode) error {
	ref := node.Ref.Data()
	action, err := p.options.Registry.GetAction(ref.Component.Name)
	if err != nil {
		return err
	}

	logger := logging.ForNode(*node)
	setupCtx := core.SetupContext{
		Configuration: node.Configuration.Data(),
		HTTP:          p.options.Registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataContext(p.tx, node),
		Requests:      contexts.NewNodeRequestContext(p.tx, node),
		Webhook:       contexts.NewNodeWebhookContext(ctx, p.tx, p.options.Encryptor, node, p.options.WebhookBaseURL),
		Auth:          contexts.NewAuthReader(p.tx, p.options.OrgID, p.options.AuthService, nil),
	}

	if node.AppInstallationID != nil {
		integration, err := models.FindUnscopedIntegrationInTransaction(p.tx, *node.AppInstallationID)
		if err != nil {
			return fmt.Errorf("failed to find app installation: %v", err)
		}

		logger = logging.WithIntegration(logger, *integration)
		setupCtx.Integration = contexts.NewIntegrationContext(
			p.tx,
			node,
			integration,
			p.options.Encryptor,
			p.options.Registry,
			nil,
		)
	}

	setupCtx.Logger = logger
	return action.Setup(setupCtx)
}

func (p *CanvasPublisher) ensureNewNodeID(node models.Node) string {

	//
	// If node ID has not been used yet, just use it.
	//
	if _, exists := p.allNodes[node.ID]; !exists {
		return node.ID
	}

	reservedIDs := make(map[string]bool)
	for _, node := range p.allNodes {
		reservedIDs[node.NodeID] = true
	}

	//
	// If node ID has been used, generate a new one.
	// Here, we update p.finalNodes to ensure that the new ID
	// is propagated to the nodes saved in the workflow_versions.nodes record.
	//
	newNodeID := models.GenerateUniqueNodeID(node, reservedIDs)
	delete(p.finalNodes, node.ID)
	node.ID = newNodeID
	p.finalNodes[newNodeID] = node
	return newNodeID
}
