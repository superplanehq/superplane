package canvases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func PublishCanvasVersion(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	title string,
	description string,
	webhookBaseURL string,
) (*pb.PublishCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	organizationUUID := uuid.MustParse(organizationID)
	userUUID := uuid.MustParse(userID)

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	if canvas.IsChangeManagementEnabled() {
		return nil, status.Error(codes.FailedPrecondition, "change management is enabled; use CreateCanvasChangeRequest instead")
	}

	requestedTitle := strings.TrimSpace(title)

	var liveVersion *models.CanvasVersion
	var draftVersion *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasInTx, findCanvasErr := models.FindCanvasInTransaction(tx, organizationUUID, canvasUUID)
		if findCanvasErr != nil {
			return findCanvasErr
		}

		if canvasInTx.LiveVersionID == nil {
			return status.Error(codes.FailedPrecondition, "canvas live version not found")
		}

		draft, findDraftErr := models.FindCanvasDraftInTransaction(tx, canvasUUID, userUUID)
		if findDraftErr != nil {
			if errors.Is(findDraftErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.FailedPrecondition, "no draft found for this user")
			}
			return findDraftErr
		}

		var findVersionErr error
		draftVersion, findVersionErr = models.FindCanvasVersionInTransaction(tx, canvasUUID, draft.VersionID)
		if findVersionErr != nil {
			if errors.Is(findVersionErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "draft version not found")
			}
			return findVersionErr
		}

		if draftVersion.OwnerID == nil || *draftVersion.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}

		if draftVersion.IsPublished {
			return status.Error(codes.FailedPrecondition, "draft is already published")
		}

		// Create a snapshot version from the draft (same as CreateCanvasChangeRequest does).
		snapshotVersion, createVersionErr := models.CreateCanvasSnapshotVersionInTransaction(
			tx,
			canvasUUID,
			userUUID,
			draftVersion.Nodes,
			draftVersion.Edges,
		)
		if createVersionErr != nil {
			return createVersionErr
		}

		now := time.Now()
		if requestedTitle == "" {
			requestedTitle = fmt.Sprintf("Published on %s", now.Format("Jan 2, 2006"))
		}

		// Create a change request that we will immediately publish.
		request := &models.CanvasChangeRequest{
			ID:               uuid.New(),
			WorkflowID:       canvasUUID,
			VersionID:        snapshotVersion.ID,
			OwnerID:          &userUUID,
			BasedOnVersionID: canvasInTx.LiveVersionID,
			Title:            requestedTitle,
			Description:      description,
			Status:           models.CanvasChangeRequestStatusOpen,
			CreatedAt:        &now,
			UpdatedAt:        &now,
		}

		if createErr := tx.Create(request).Error; createErr != nil {
			return createErr
		}

		if diffErr := refreshCanvasChangeRequestDiffInTransaction(tx, canvasInTx, snapshotVersion, request); diffErr != nil {
			return diffErr
		}

		if len(request.ConflictingNodeIDs) > 0 {
			return status.Error(codes.FailedPrecondition, "draft has conflicts with the live canvas; resolve conflicts before publishing")
		}

		baseNodes, baseEdges, liveNodes, liveEdges, resolveErr := resolveCanvasChangeRequestBaseAndLiveInTransaction(
			tx,
			canvasInTx,
			request,
		)
		if resolveErr != nil {
			return resolveErr
		}

		mergedNodes, mergedEdges := mergeCanvasVersionIntoLive(
			baseNodes,
			baseEdges,
			liveNodes,
			liveEdges,
			snapshotVersion.Nodes,
			snapshotVersion.Edges,
			request.ChangedNodeIDs,
		)

		existingNodesUnscoped, findNodesErr := models.FindCanvasNodesUnscopedInTransaction(tx, canvasUUID)
		if findNodesErr != nil {
			return findNodesErr
		}

		mergedNodes, mergedEdges, _ = remapNodeIDsForConflicts(canvasUUID, mergedNodes, mergedEdges, existingNodesUnscoped)

		parentNodesByNodeID := make(map[string]*models.Node)
		for i := range mergedNodes {
			parentNodesByNodeID[mergedNodes[i].ID] = &mergedNodes[i]
		}

		expandedNodes, expandErr := expandNodes(organizationID, mergedNodes)
		if expandErr != nil {
			return expandErr
		}

		existingNodes, findErr := models.FindCanvasNodesInTransaction(tx, canvasUUID)
		if findErr != nil {
			return findErr
		}

		for _, node := range expandedNodes {
			if node.Type == models.NodeTypeWidget {
				continue
			}

			workflowNode, nodeLevelErrorMessage, upsertErr := upsertNode(tx, existingNodes, node, canvasUUID)
			if upsertErr != nil {
				return upsertErr
			}

			if nodeLevelErrorMessage != nil {
				errorNodeID := node.ID
				if workflowNode.ParentNodeID != nil {
					errorNodeID = *workflowNode.ParentNodeID
				}
				parentNode, ok := parentNodesByNodeID[errorNodeID]
				if !ok {
					log.Errorf("Parent node %s not found for node-level error", errorNodeID)
				} else {
					parentNode.ErrorMessage = nodeLevelErrorMessage
				}
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
			&userUUID,
			mergedNodes,
			mergedEdges,
		)
		if err != nil {
			return err
		}

		canvasInTx.LiveVersionID = &liveVersion.ID
		canvasInTx.UpdatedAt = liveVersion.UpdatedAt

		request.Status = models.CanvasChangeRequestStatusPublished
		request.PublishedAt = &now
		request.UpdatedAt = &now
		if saveErr := tx.Save(request).Error; saveErr != nil {
			return saveErr
		}

		if refreshErr := refreshOpenCanvasChangeRequestsInTransaction(tx, organizationUUID, canvasUUID, request.ID); refreshErr != nil {
			return refreshErr
		}

		// Delete the user's draft — it has been published and is no longer needed.
		if deleteErr := tx.Delete(&models.CanvasUserDraft{}, "workflow_id = ? AND user_id = ?", canvasUUID, userUUID).Error; deleteErr != nil {
			return deleteErr
		}

		canvas = canvasInTx
		return nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, actions.ToStatus(err)
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}
	if liveVersion != nil {
		if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), liveVersion.ID.String()).PublishVersionUpdated(); err != nil {
			log.Errorf("failed to publish canvas version updated RabbitMQ message: %v", err)
		}
	}
	if draftVersion != nil {
		if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), draftVersion.ID.String()).PublishVersionUpdated(); err != nil {
			log.Errorf("failed to publish canvas version updated RabbitMQ message: %v", err)
		}
	}

	return &pb.PublishCanvasVersionResponse{
		Version: SerializeCanvasVersion(liveVersion, organizationID),
	}, nil
}
