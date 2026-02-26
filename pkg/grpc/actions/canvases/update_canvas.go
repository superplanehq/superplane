package canvases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func UpdateCanvas(ctx context.Context, encryptor crypto.Encryptor, registry *registry.Registry, organizationID string, id string, pbCanvas *pb.Canvas, webhookBaseURL string) (*pb.UpdateCanvasResponse, error) {
	return UpdateCanvasWithAutoLayout(ctx, encryptor, registry, organizationID, id, pbCanvas, nil, webhookBaseURL)
}

func UpdateCanvasWithAutoLayout(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	id string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
	webhookBaseURL string,
) (*pb.UpdateCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	existingCanvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if _, templateErr := models.FindCanvasTemplate(canvasID); templateErr == nil {
				return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
			}
		}
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if existingCanvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	nodes, edges, err := ParseCanvas(registry, organizationID, pbCanvas)
	if err != nil {
		return nil, actions.ToStatus(err)
	}

	nodes, edges, err = applyCanvasAutoLayout(nodes, edges, autoLayout)
	if err != nil {
		return nil, actions.ToStatus(err)
	}

	existingNodesUnscoped, err := models.FindCanvasNodesUnscoped(canvasID)
	if err != nil {
		return nil, actions.ToStatus(err)
	}

	nodes, edges, _ = remapNodeIDsForConflicts(nodes, edges, existingNodesUnscoped)

	parentNodesByNodeID := make(map[string]*models.Node)
	for i := range nodes {
		parentNodesByNodeID[nodes[i].ID] = &nodes[i]
	}

	expandedNodes, err := expandNodes(organizationID, nodes)
	if err != nil {
		return nil, actions.ToStatus(err)
	}

	now := time.Now()

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		//
		// Update the canvas node records
		//
		existingNodes, err := models.FindCanvasNodesInTransaction(tx, existingCanvas.ID)
		if err != nil {
			return err
		}

		//
		// Go through each node in the new canvas, creating / updating it,
		// and tracking which nodes we've seen, to delete nodes that are no longer in the workflow at the end.
		//
		for _, node := range expandedNodes {
			// Widgets are not persisted in workflow_nodes table and don't have any logic to execute and to setup.
			if node.Type == models.NodeTypeWidget {
				continue
			}

			workflowNode, err := upsertNode(tx, existingNodes, node, canvasID)
			if err != nil {
				return err
			}

			if workflowNode.State == models.CanvasNodeStateReady {
				err = setupNode(ctx, tx, encryptor, registry, workflowNode, webhookBaseURL)
				if err != nil {
					workflowNode.State = models.CanvasNodeStateError
					errorMsg := err.Error()
					workflowNode.StateReason = &errorMsg
					if saveErr := tx.Save(workflowNode).Error; saveErr != nil {
						return saveErr
					}

					errorNodeID := node.ID
					if workflowNode.ParentNodeID != nil {
						errorNodeID = *workflowNode.ParentNodeID
					}

					parentNode, ok := parentNodesByNodeID[errorNodeID]
					if !ok {
						log.Errorf("Parent node %s not found for node setup error", errorNodeID)
					} else {
						parentNode.ErrorMessage = &errorMsg
					}
				}

				if workflowNode.ParentNodeID == nil {
					parentNode, ok := parentNodesByNodeID[workflowNode.NodeID]
					if !ok {
						log.Errorf("Parent node %s not found", workflowNode.NodeID)
						return status.Errorf(codes.Internal, "It was not possible to find the parent node %s", workflowNode.NodeID)
					}
					parentNode.Metadata = workflowNode.Metadata.Data()
				}
			}
		}

		//
		// Update the workflow record latest because we need to setup the metadata of the parent nodes
		//
		existingCanvas.Name = pbCanvas.Metadata.Name
		existingCanvas.Description = pbCanvas.Metadata.Description
		existingCanvas.UpdatedAt = &now
		existingCanvas.Edges = datatypes.NewJSONSlice(edges)
		existingCanvas.Nodes = datatypes.NewJSONSlice(nodes)
		err = tx.Save(&existingCanvas).Error
		if err != nil {
			return err
		}

		return deleteNodes(tx, existingNodes, expandedNodes)
	})

	if err != nil {
		return nil, actions.ToStatus(err)
	}

	protoCanvas, err := SerializeCanvas(existingCanvas, true)
	if err != nil {
		return nil, actions.ToStatus(err)
	}

	if err := messages.NewCanvasUpdatedMessage(existingCanvas.ID.String()).Publish(true); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}

	return &pb.UpdateCanvasResponse{
		Canvas: protoCanvas,
	}, nil
}

// Remap node IDs that conflict with soft-deleted workflow_nodes entries so we
// can preserve historical records while still allowing new nodes with similar
// names to be created in the same workflow.
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

	//
	// Node exists, just update it
	//
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

		// Set parent if internal namespaced id
		if idx := strings.Index(node.ID, ":"); idx != -1 {
			parent := node.ID[:idx]
			existingNode.ParentNodeID = &parent
		} else {
			existingNode.ParentNodeID = nil
		}

		existingNode.UpdatedAt = &now
		err := tx.Save(&existingNode).Error
		if err != nil {
			return nil, err
		}

		return existingNode, nil
	}

	//
	// Node doesn't exist, create it
	//
	// Derive ParentNodeID for internal nodes
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

	err := tx.Create(&canvasNode).Error
	if err != nil {
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
		// Widgets are not persisted and don't have any logic to execute and to setup.
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
		Events:        contexts.NewEventContext(tx, node),
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
		)
	}

	triggerCtx.Logger = logger
	err = trigger.Setup(triggerCtx)
	if err != nil {
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
		)
	}

	setupCtx.Logger = logger
	err = component.Setup(setupCtx)
	if err != nil {
		return fmt.Errorf("error setting up node %s: %v", node.NodeID, err)
	}

	return tx.Save(node).Error
}

func deleteNodes(tx *gorm.DB, existingNodes []models.CanvasNode, newNodes []models.Node) error {
	for _, existingNode := range existingNodes {
		if !slices.ContainsFunc(newNodes, func(n models.Node) bool { return n.ID == existingNode.NodeID }) {
			err := models.DeleteCanvasNode(tx, existingNode)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
