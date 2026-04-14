package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// applyCanvasSpecInTransaction remaps node IDs, expands blueprint nodes,
// upserts/sets-up every node, and deletes removed nodes.
// The input nodes slice is mutated in place (ErrorMessage, Metadata fields).
func applyCanvasSpecInTransaction(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	organizationID string,
	organizationUUID uuid.UUID,
	canvasID uuid.UUID,
	nodes []models.Node,
	edges []models.Edge,
	authService authorization.Authorization,
	webhookBaseURL string,
) ([]models.Node, []models.Edge, error) {
	existingNodesUnscoped, err := models.FindCanvasNodesUnscopedInTransaction(tx, canvasID)
	if err != nil {
		return nil, nil, err
	}

	nodes, edges, _ = remapNodeIDsForConflicts(canvasID, nodes, edges, existingNodesUnscoped)

	parentNodesByNodeID := make(map[string]*models.Node, len(nodes))
	for i := range nodes {
		parentNodesByNodeID[nodes[i].ID] = &nodes[i]
	}

	expandedNodes, err := expandNodes(organizationID, nodes)
	if err != nil {
		return nil, nil, err
	}

	existingNodes, err := models.FindCanvasNodesInTransaction(tx, canvasID)
	if err != nil {
		return nil, nil, err
	}

	for _, node := range expandedNodes {
		if node.Type == models.NodeTypeWidget {
			continue
		}

		if err := applyNode(ctx, tx, encryptor, reg, organizationUUID, canvasID, existingNodes, node, parentNodesByNodeID, authService, webhookBaseURL); err != nil {
			return nil, nil, err
		}
	}

	if err := deleteNodes(tx, existingNodes, expandedNodes); err != nil {
		return nil, nil, err
	}

	return nodes, edges, nil
}

func applyNode(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	organizationUUID uuid.UUID,
	canvasID uuid.UUID,
	existingNodes []models.CanvasNode,
	node models.Node,
	parentNodesByNodeID map[string]*models.Node,
	authService authorization.Authorization,
	webhookBaseURL string,
) error {
	workflowNode, nodeLevelErrorMessage, err := upsertNode(tx, existingNodes, node, canvasID)
	if err != nil {
		return err
	}

	if nodeLevelErrorMessage != nil {
		setParentNodeError(workflowNode, node.ID, parentNodesByNodeID, nodeLevelErrorMessage)
	}

	if workflowNode.State == models.CanvasNodeStateReady {
		if err := setupNode(ctx, tx, encryptor, reg, workflowNode, organizationUUID, authService, webhookBaseURL); err != nil {
			if saveErr := markNodeSetupError(tx, workflowNode, err); saveErr != nil {
				return saveErr
			}

			errorMsg := err.Error()
			setParentNodeError(workflowNode, node.ID, parentNodesByNodeID, &errorMsg)
		}
	}

	if workflowNode.ParentNodeID != nil {
		return nil
	}

	parentNode, exists := parentNodesByNodeID[workflowNode.NodeID]
	if !exists {
		log.Errorf("parent node %s not found", workflowNode.NodeID)
		return status.Errorf(codes.Internal, "it was not possible to find the parent node %s", workflowNode.NodeID)
	}
	parentNode.Metadata = workflowNode.Metadata.Data()

	return nil
}

func setParentNodeError(workflowNode *models.CanvasNode, nodeID string, parentNodesByNodeID map[string]*models.Node, errorMessage *string) {
	errorNodeID := nodeID
	if workflowNode.ParentNodeID != nil {
		errorNodeID = *workflowNode.ParentNodeID
	}
	if parentNode, ok := parentNodesByNodeID[errorNodeID]; ok {
		parentNode.ErrorMessage = errorMessage
	}
}

func markNodeSetupError(tx *gorm.DB, workflowNode *models.CanvasNode, setupErr error) error {
	workflowNode.State = models.CanvasNodeStateError
	errorMsg := setupErr.Error()
	workflowNode.StateReason = &errorMsg
	return tx.Save(workflowNode).Error
}
