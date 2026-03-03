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
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func PublishCanvasChangeRequest(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	changeRequestID string,
	expectedLiveVersionID string,
	webhookBaseURL string,
) (*pb.PublishCanvasChangeRequestResponse, error) {
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}
	organizationUUID := uuid.MustParse(organizationID)

	changeRequestUUID, err := uuid.Parse(changeRequestID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid change request id: %v", err)
	}

	var expectedLiveVersionUUID *uuid.UUID
	if expectedLiveVersionID != "" {
		parsedExpected, parseErr := uuid.Parse(expectedLiveVersionID)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid expected live version id: %v", parseErr)
		}
		expectedLiveVersionUUID = &parsedExpected
	}

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	var version *models.CanvasVersion
	var request *models.CanvasChangeRequest

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
			return status.Error(codes.FailedPrecondition, "change request was already published")
		}
		if request.Status == models.CanvasChangeRequestStatusClosed {
			return status.Error(codes.FailedPrecondition, "change request is closed")
		}

		version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, request.VersionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return err
		}

		if expectedLiveVersionUUID != nil {
			if canvasForUpdate.LiveVersionID == nil || *canvasForUpdate.LiveVersionID != *expectedLiveVersionUUID {
				return status.Error(codes.FailedPrecondition, "live version changed")
			}
		}

		if err := refreshCanvasChangeRequestDiffInTransaction(tx, canvasForUpdate, version, request); err != nil {
			return err
		}

		if len(request.ConflictingNodeIDs) > 0 {
			return status.Error(codes.FailedPrecondition, "change request has conflicts")
		}

		baseNodes, baseEdges, liveNodes, liveEdges, resolveErr := resolveCanvasVersionBaseAndLiveInTransaction(tx, canvasForUpdate, version)
		if resolveErr != nil {
			return resolveErr
		}

		mergedNodes, mergedEdges := mergeCanvasVersionIntoLive(
			baseNodes,
			baseEdges,
			liveNodes,
			liveEdges,
			version.Nodes,
			version.Edges,
			request.ChangedNodeIDs,
		)

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

		canvasForUpdate.UpdatedAt = &now
		canvasForUpdate.Nodes = datatypes.NewJSONSlice(mergedNodes)
		canvasForUpdate.Edges = datatypes.NewJSONSlice(mergedEdges)
		canvasForUpdate.LiveVersionID = &version.ID
		if saveErr := tx.Save(canvasForUpdate).Error; saveErr != nil {
			return saveErr
		}

		if deleteErr := deleteNodes(tx, existingNodes, expandedNodes); deleteErr != nil {
			return deleteErr
		}

		version.Nodes = datatypes.NewJSONSlice(mergedNodes)
		version.Edges = datatypes.NewJSONSlice(mergedEdges)
		version.IsPublished = true
		version.PublishedAt = &now
		version.UpdatedAt = &now
		if saveErr := tx.Save(version).Error; saveErr != nil {
			return saveErr
		}

		request.Status = models.CanvasChangeRequestStatusPublished
		request.PublishedAt = &now
		request.UpdatedAt = &now
		if saveErr := tx.Save(request).Error; saveErr != nil {
			return saveErr
		}

		if deleteErr := tx.Where("workflow_id = ? AND version_id = ?", canvasUUID, version.ID).Delete(&models.CanvasUserDraft{}).Error; deleteErr != nil {
			return deleteErr
		}

		if refreshErr := refreshOpenCanvasChangeRequestsInTransaction(tx, organizationUUID, canvasUUID, request.ID); refreshErr != nil {
			return refreshErr
		}

		canvas = canvasForUpdate
		return nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, actions.ToStatus(err)
	}

	protoCanvas, err := SerializeCanvas(canvas, true)
	if err != nil {
		return nil, actions.ToStatus(err)
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String()).Publish(true); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}
	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas version updated RabbitMQ message: %v", err)
	}

	return &pb.PublishCanvasChangeRequestResponse{
		Canvas:        protoCanvas,
		Version:       SerializeCanvasVersion(version, organizationID),
		ChangeRequest: SerializeCanvasChangeRequest(request, version, organizationID),
	}, nil
}
