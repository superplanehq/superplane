package workflows

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func UpdateWorkflow(ctx context.Context, encryptor crypto.Encryptor, registry *registry.Registry, organizationID string, id string, pbWorkflow *pb.Workflow, webhookBaseURL string) (*pb.UpdateWorkflowResponse, error) {
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow id: %v", err)
	}

	existingWorkflow, err := models.FindWorkflow(uuid.MustParse(organizationID), workflowID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	nodes, edges, err := ParseWorkflow(registry, organizationID, pbWorkflow)
	if err != nil {
		return nil, err
	}

	expandedNodes, err := expandNodes(organizationID, nodes)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		//
		// Update the workflow record first
		//
		existingWorkflow.Name = pbWorkflow.Metadata.Name
		existingWorkflow.Description = pbWorkflow.Metadata.Description
		existingWorkflow.UpdatedAt = &now
		existingWorkflow.Edges = datatypes.NewJSONSlice(edges)
		err := tx.Save(&existingWorkflow).Error
		if err != nil {
			return err
		}

		//
		// Update the workflow node records
		//
		existingNodes, err := models.FindWorkflowNodesInTransaction(tx, existingWorkflow.ID)
		if err != nil {
			return err
		}

		//
		// Go through each node in the new workflow, creating / updating it,
		// and tracking which nodes we've seen, to delete nodes that are no longer in the workflow at the end.
		//
		for _, node := range expandedNodes {
			workflowNode, err := upsertNode(tx, existingNodes, node, workflowID)
			if err != nil {
				return err
			}

			// Skip setup for annotation nodes - they are text-only and don't participate in workflow execution
			if workflowNode.Type == models.NodeTypeAnnotation {
				continue
			}

			if workflowNode.State == models.WorkflowNodeStateReady {
				err = setupNode(ctx, tx, encryptor, registry, *workflowNode, webhookBaseURL)
				if err != nil {
					workflowNode.State = models.WorkflowNodeStateError
					errorMsg := err.Error()
					workflowNode.StateReason = &errorMsg
					if saveErr := tx.Save(workflowNode).Error; saveErr != nil {
						return saveErr
					}
				}
			}
		}

		return deleteNodes(tx, existingNodes, expandedNodes)
	})

	if err != nil {
		return nil, err
	}

	protoWorkflow, err := SerializeWorkflow(existingWorkflow, true)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateWorkflowResponse{
		Workflow: protoWorkflow,
	}, nil
}

func findNode(nodes []models.WorkflowNode, nodeID string) *models.WorkflowNode {
	for _, node := range nodes {
		if node.NodeID == nodeID {
			return &node
		}
	}
	return nil
}

func upsertNode(tx *gorm.DB, existingNodes []models.WorkflowNode, node models.Node, workflowID uuid.UUID) (*models.WorkflowNode, error) {
	now := time.Now()

	var appInstallationID *uuid.UUID
	if node.AppInstallationID != nil && *node.AppInstallationID != "" {
		parsedID, err := uuid.Parse(*node.AppInstallationID)
		if err != nil {
			return nil, fmt.Errorf("invalid app installation ID: %v", err)
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
			existingNode.State = models.WorkflowNodeStateError
			existingNode.StateReason = node.ErrorMessage
			existingNode.AnnotationText = nil
		} else if existingNode.State == models.WorkflowNodeStateError && node.Type != models.NodeTypeAnnotation {
			existingNode.State = models.WorkflowNodeStateReady
			existingNode.StateReason = nil
			existingNode.AnnotationText = node.AnnotationText
		} else if node.Type == models.NodeTypeAnnotation && existingNode.State != models.WorkflowNodeStateStatic {
			existingNode.State = models.WorkflowNodeStateStatic
			existingNode.AnnotationText = node.AnnotationText
		} else {
			existingNode.AnnotationText = node.AnnotationText
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

	initialState := models.WorkflowNodeStateReady
	var stateReason *string
	var annotationText *string

	if node.ErrorMessage != nil && *node.ErrorMessage != "" {
		initialState = models.WorkflowNodeStateError
		stateReason = node.ErrorMessage
		annotationText = nil
	} else if node.Type == models.NodeTypeAnnotation {
		initialState = models.WorkflowNodeStateStatic
		annotationText = node.AnnotationText
	} else {
		annotationText = node.AnnotationText
	}

	workflowNode := models.WorkflowNode{
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
		AnnotationText:    annotationText,
		AppInstallationID: appInstallationID,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}

	err := tx.Create(&workflowNode).Error
	if err != nil {
		return nil, err
	}

	return &workflowNode, nil
}

func setupNode(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, registry *registry.Registry, node models.WorkflowNode, webhookBaseURL string) error {
	switch node.Type {
	case models.NodeTypeTrigger:
		return setupTrigger(ctx, tx, encryptor, registry, node, webhookBaseURL)
	case models.NodeTypeComponent:
		return setupComponent(tx, encryptor, registry, node)
	case models.NodeTypeAnnotation:
		return nil
	}

	return nil
}

func setupTrigger(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, registry *registry.Registry, node models.WorkflowNode, webhookBaseURL string) error {
	ref := node.Ref.Data()
	trigger, err := registry.GetTrigger(ref.Trigger.Name)
	if err != nil {
		return err
	}

	logger := logging.ForNode(node)
	triggerCtx := core.TriggerContext{
		Configuration:      node.Configuration.Data(),
		MetadataContext:    contexts.NewNodeMetadataContext(tx, &node),
		RequestContext:     contexts.NewNodeRequestContext(tx, &node),
		IntegrationContext: contexts.NewIntegrationContext(tx, registry),
		EventContext:       contexts.NewEventContext(tx, &node),
		WebhookContext:     contexts.NewWebhookContext(ctx, tx, encryptor, &node, webhookBaseURL),
	}

	if node.AppInstallationID != nil {
		appInstallation, err := models.FindUnscopedAppInstallationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			return fmt.Errorf("failed to find app installation: %v", err)
		}

		logger = logging.WithAppInstallation(logger, *appInstallation)
		triggerCtx.AppInstallationContext = contexts.NewAppInstallationContext(
			tx,
			&node,
			appInstallation,
			encryptor,
			registry,
		)
	}

	triggerCtx.Logger = logger
	err = trigger.Setup(triggerCtx)
	if err != nil {
		return fmt.Errorf("error setting up node %s: %v", node.NodeID, err)
	}

	return tx.Save(&node).Error
}

func setupComponent(tx *gorm.DB, encryptor crypto.Encryptor, registry *registry.Registry, node models.WorkflowNode) error {
	ref := node.Ref.Data()
	component, err := registry.GetComponent(ref.Component.Name)
	if err != nil {
		return err
	}

	logger := logging.ForNode(node)
	setupCtx := core.SetupContext{
		Configuration:      node.Configuration.Data(),
		MetadataContext:    contexts.NewNodeMetadataContext(tx, &node),
		RequestContext:     contexts.NewNodeRequestContext(tx, &node),
		IntegrationContext: contexts.NewIntegrationContext(tx, registry),
	}

	if node.AppInstallationID != nil {
		appInstallation, err := models.FindUnscopedAppInstallationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			return fmt.Errorf("failed to find app installation: %v", err)
		}

		logger = logging.WithAppInstallation(logger, *appInstallation)
		setupCtx.AppInstallationContext = contexts.NewAppInstallationContext(
			tx,
			&node,
			appInstallation,
			encryptor,
			registry,
		)
	}

	setupCtx.Logger = logger
	err = component.Setup(setupCtx)
	if err != nil {
		return fmt.Errorf("error setting up node %s: %v", node.NodeID, err)
	}

	return tx.Save(&node).Error
}

func deleteNodes(tx *gorm.DB, existingNodes []models.WorkflowNode, newNodes []models.Node) error {
	for _, existingNode := range existingNodes {
		if !slices.ContainsFunc(newNodes, func(n models.Node) bool { return n.ID == existingNode.NodeID }) {
			err := models.DeleteWorkflowNode(tx, existingNode)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
