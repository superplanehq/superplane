package canvases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func PublishCanvasChangeRequest(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	changeRequestID string,
	webhookBaseURL string,
) (*models.CanvasChangeRequest, *models.CanvasVersion, error) {
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}
	organizationUUID := uuid.MustParse(organizationID)

	changeRequestUUID, err := uuid.Parse(changeRequestID)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid change request id: %v", err)
	}

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	sandboxModeEnabled, modeErr := isCanvasSandboxModeEnabled(organizationID)
	if modeErr != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to load organization sandbox mode: %v", modeErr)
	}
	if sandboxModeEnabled {
		return nil, nil, status.Error(codes.FailedPrecondition, "canvas versioning is disabled in sandbox mode")
	}

	var version *models.CanvasVersion
	var request *models.CanvasChangeRequest
	var liveVersion *models.CanvasVersion
	var renewedDraftVersion *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasForUpdate, canvasErr := models.FindCanvasInTransaction(tx, organizationUUID, canvasUUID)
		if canvasErr != nil {
			return canvasErr
		}

		request, err = models.FindCanvasChangeRequestInTransaction(tx, canvasUUID, changeRequestUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "change request not found")
			}
			return err
		}

		if request.Status == models.CanvasChangeRequestStatusPublished {
			return status.Error(codes.FailedPrecondition, "change request was already merged")
		}

		version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, request.VersionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return err
		}

		mergedNodes := append([]models.Node(nil), version.Nodes...)
		mergedEdges := append([]models.Edge(nil), version.Edges...)

		existingNodesUnscoped, findNodesErr := models.FindCanvasNodesUnscopedInTransaction(tx, canvasUUID)
		if findNodesErr != nil {
			return findNodesErr
		}

		mergedNodes, mergedEdges, _ = remapNodeIDsForConflicts(mergedNodes, mergedEdges, existingNodesUnscoped)

		parentNodesByNodeID := make(map[string]*models.Node)
		for i := range mergedNodes {
			parentNodesByNodeID[mergedNodes[i].ID] = &mergedNodes[i]
		}

		expandedNodes, expandErr := expandNodes(organizationID, mergedNodes)
		if expandErr != nil {
			return expandErr
		}

		now := time.Now()

		existingNodes, findErr := models.FindCanvasNodesInTransaction(tx, canvasUUID)
		if findErr != nil {
			return findErr
		}

		for _, node := range expandedNodes {
			if node.Type == models.NodeTypeWidget {
				continue
			}

			workflowNode, upsertErr := upsertNode(tx, existingNodes, node, canvasUUID)
			if upsertErr != nil {
				return upsertErr
			}

			if workflowNode.State == models.CanvasNodeStateReady {
				setupErr := setupNode(ctx, tx, encryptor, registry, workflowNode, webhookBaseURL)
				if setupErr != nil {
					workflowNode.State = models.CanvasNodeStateError
					errorMsg := setupErr.Error()
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
			}

			if workflowNode.ParentNodeID != nil {
				continue
			}

			parentNode, exists := parentNodesByNodeID[workflowNode.NodeID]
			if !exists {
				log.Errorf("Parent node %s not found", workflowNode.NodeID)
				return status.Errorf(codes.Internal, "It was not possible to find the parent node %s", workflowNode.NodeID)
			}
			parentNode.Metadata = workflowNode.Metadata.Data()
		}

		if deleteErr := deleteNodes(tx, existingNodes, expandedNodes); deleteErr != nil {
			return deleteErr
		}

		liveVersion, err = models.CreatePublishedCanvasVersionInTransaction(
			tx,
			canvasUUID,
			request.OwnerID,
			mergedNodes,
			mergedEdges,
		)
		if err != nil {
			return err
		}
		canvasForUpdate.LiveVersionID = &liveVersion.ID
		canvasForUpdate.UpdatedAt = liveVersion.UpdatedAt

		request.Status = models.CanvasChangeRequestStatusPublished
		request.PublishedAt = &now
		request.UpdatedAt = &now
		if saveErr := tx.Save(request).Error; saveErr != nil {
			return saveErr
		}

		if request.OwnerID != nil {
			renewedDraftVersion, err = models.SaveCanvasDraftInTransaction(
				tx,
				canvasUUID,
				*request.OwnerID,
				liveVersion.Nodes,
				liveVersion.Edges,
			)
			if err != nil {
				return err
			}
		}

		canvas = canvasForUpdate
		return nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, nil, err
		}
		return nil, nil, actions.ToStatus(err)
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String()).Publish(true); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}
	if liveVersion != nil {
		if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), liveVersion.ID.String()).PublishVersionUpdated(); err != nil {
			log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
		}
	}
	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}
	if renewedDraftVersion != nil {
		if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), renewedDraftVersion.ID.String()).PublishVersionUpdated(); err != nil {
			log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
		}
	}

	return request, version, nil
}
