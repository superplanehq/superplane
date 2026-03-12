package canvases

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// remapNodeIDsForConflicts avoids collisions with soft-deleted workflow_nodes
// while preserving old node records for historical data.
func remapNodeIDsForConflicts(
	nodes []models.Node,
	edges []models.Edge,
	existingNodes []models.CanvasNode,
) ([]models.Node, []models.Edge, map[string]string) {
	reservedIDs := make(map[string]bool, len(existingNodes))
	deletedIDs := make(map[string]bool, len(existingNodes))

	for _, existing := range existingNodes {
		reservedIDs[existing.NodeID] = true
		if existing.DeletedAt.Valid {
			deletedIDs[existing.NodeID] = true
		}
	}

	remappedIDs := map[string]string{}
	for i := range nodes {
		if !deletedIDs[nodes[i].ID] {
			reservedIDs[nodes[i].ID] = true
			continue
		}

		newID := models.GenerateUniqueNodeID(nodes[i], reservedIDs)
		remappedIDs[nodes[i].ID] = newID
		nodes[i].ID = newID
		reservedIDs[newID] = true
	}

	if len(remappedIDs) == 0 {
		return nodes, edges, remappedIDs
	}

	for i := range edges {
		if newID, ok := remappedIDs[edges[i].SourceID]; ok {
			edges[i].SourceID = newID
		}
		if newID, ok := remappedIDs[edges[i].TargetID]; ok {
			edges[i].TargetID = newID
		}
	}

	return nodes, edges, remappedIDs
}

func findNode(nodes []models.CanvasNode, nodeID string) *models.CanvasNode {
	for _, node := range nodes {
		if node.NodeID == nodeID {
			return &node
		}
	}
	return nil
}

func upsertNode(tx *gorm.DB, existingNodes []models.CanvasNode, node models.Node, workflowID uuid.UUID) (*models.CanvasNode, error) {
	now := time.Now()

	var appInstallationID *uuid.UUID
	if node.IntegrationID != nil && *node.IntegrationID != "" {
		parsedID, err := uuid.Parse(*node.IntegrationID)
		if err != nil {
			return nil, fmt.Errorf("invalid integration ID: %v", err)
		}
		appInstallationID = &parsedID
	}

	existingNode := findNode(existingNodes, node.ID)
	if existingNode != nil {
		existingNode.Name = node.Name
		existingNode.Type = node.Type
		existingNode.Ref = datatypes.NewJSONType(node.Ref)
		existingNode.Configuration = datatypes.NewJSONType(node.Configuration)
		existingNode.Position = datatypes.NewJSONType(node.Position)
		existingNode.IsCollapsed = node.IsCollapsed
		existingNode.AppInstallationID = appInstallationID

		if node.ErrorMessage != nil && *node.ErrorMessage != "" {
			existingNode.State = models.CanvasNodeStateError
			existingNode.StateReason = node.ErrorMessage
		} else if existingNode.State == models.CanvasNodeStateError {
			existingNode.State = models.CanvasNodeStateReady
			existingNode.StateReason = nil
		}

		if idx := strings.Index(node.ID, ":"); idx != -1 {
			parent := node.ID[:idx]
			existingNode.ParentNodeID = &parent
		} else {
			existingNode.ParentNodeID = nil
		}

		existingNode.UpdatedAt = &now
		if err := tx.Save(&existingNode).Error; err != nil {
			return nil, err
		}

		return existingNode, nil
	}

	var parentNodeID *string
	if idx := strings.Index(node.ID, ":"); idx != -1 {
		parent := node.ID[:idx]
		parentNodeID = &parent
	}

	initialState := models.CanvasNodeStateReady
	var stateReason *string
	if node.ErrorMessage != nil && *node.ErrorMessage != "" {
		initialState = models.CanvasNodeStateError
		stateReason = node.ErrorMessage
	}

	canvasNode := models.CanvasNode{
		WorkflowID:        workflowID,
		NodeID:            node.ID,
		ParentNodeID:      parentNodeID,
		Name:              node.Name,
		State:             initialState,
		StateReason:       stateReason,
		Type:              node.Type,
		Ref:               datatypes.NewJSONType(node.Ref),
		Configuration:     datatypes.NewJSONType(node.Configuration),
		Position:          datatypes.NewJSONType(node.Position),
		IsCollapsed:       node.IsCollapsed,
		Metadata:          datatypes.NewJSONType(node.Metadata),
		AppInstallationID: appInstallationID,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}

	if err := tx.Create(&canvasNode).Error; err != nil {
		return nil, err
	}

	return &canvasNode, nil
}

func setupNode(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, registry *registry.Registry, node *models.CanvasNode, webhookBaseURL string) error {
	switch node.Type {
	case models.NodeTypeTrigger:
		return setupTrigger(ctx, tx, encryptor, registry, node, webhookBaseURL)
	case models.NodeTypeComponent:
		return setupComponent(ctx, tx, encryptor, registry, node, webhookBaseURL)
	case models.NodeTypeWidget:
		return nil
	}

	return nil
}

func setupTrigger(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, registry *registry.Registry, node *models.CanvasNode, webhookBaseURL string) error {
	ref := node.Ref.Data()
	trigger, err := registry.GetTrigger(ref.Trigger.Name)
	if err != nil {
		return err
	}

	logger := logging.ForNode(*node)
	triggerCtx := core.TriggerContext{
		Configuration: node.Configuration.Data(),
		HTTP:          registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataContext(tx, node),
		Requests:      contexts.NewNodeRequestContext(tx, node),
		Events:        contexts.NewEventContext(tx, node, nil),
		Webhook:       contexts.NewNodeWebhookContext(ctx, tx, encryptor, node, webhookBaseURL),
	}

	if node.AppInstallationID != nil {
		integration, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			return fmt.Errorf("failed to find app installation: %v", err)
		}

		logger = logging.WithIntegration(logger, *integration)
		triggerCtx.Integration = contexts.NewIntegrationContext(
			tx,
			node,
			integration,
			encryptor,
			registry,
			nil,
		)
	}

	triggerCtx.Logger = logger
	if err := trigger.Setup(triggerCtx); err != nil {
		return fmt.Errorf("error setting up node %s: %v", node.NodeID, err)
	}

	return tx.Save(node).Error
}

func setupComponent(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, registry *registry.Registry, node *models.CanvasNode, webhookBaseURL string) error {
	ref := node.Ref.Data()
	component, err := registry.GetComponent(ref.Component.Name)
	if err != nil {
		return err
	}

	logger := logging.ForNode(*node)
	setupCtx := core.SetupContext{
		Configuration: node.Configuration.Data(),
		HTTP:          registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataContext(tx, node),
		Requests:      contexts.NewNodeRequestContext(tx, node),
		Webhook:       contexts.NewNodeWebhookContext(ctx, tx, encryptor, node, webhookBaseURL),
	}

	if node.AppInstallationID != nil {
		integration, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			return fmt.Errorf("failed to find app installation: %v", err)
		}

		logger = logging.WithIntegration(logger, *integration)
		setupCtx.Integration = contexts.NewIntegrationContext(
			tx,
			node,
			integration,
			encryptor,
			registry,
			nil,
		)
	}

	setupCtx.Logger = logger
	if err := component.Setup(setupCtx); err != nil {
		return fmt.Errorf("error setting up node %s: %v", node.NodeID, err)
	}

	return tx.Save(node).Error
}

func deleteNodes(tx *gorm.DB, existingNodes []models.CanvasNode, newNodes []models.Node) error {
	for _, existingNode := range existingNodes {
		if !slices.ContainsFunc(newNodes, func(n models.Node) bool { return n.ID == existingNode.NodeID }) {
			if err := models.DeleteCanvasNode(tx, existingNode); err != nil {
				return err
			}
		}
	}

	return nil
}
