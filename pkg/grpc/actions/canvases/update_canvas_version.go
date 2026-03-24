package canvases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func UpdateCanvasVersion(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
	webhookBaseURL string,
) (*pb.UpdateCanvasVersionResponse, error) {
	return UpdateCanvasVersionWithUsage(
		ctx,
		nil,
		encryptor,
		registry,
		organizationID,
		canvasID,
		versionID,
		pbCanvas,
		autoLayout,
		webhookBaseURL,
	)
}

func UpdateCanvasVersionWithUsage(
	ctx context.Context,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
	webhookBaseURL string,
) (*pb.UpdateCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}
	organizationUUID := uuid.MustParse(organizationID)

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	nodes, edges, err := ParseCanvas(registry, organizationID, pbCanvas)
	if err != nil {
		return nil, err
	}

	nodes, edges, err = applyCanvasAutoLayout(nodes, edges, autoLayout, registry)
	if err != nil {
		return nil, err
	}

	expandedNodes, err := expandNodes(organizationID, nodes)
	if err != nil {
		return nil, err
	}

	if err := usage.EnsureOrganizationWithinLimits(ctx, usageService, organizationID, &usagepb.OrganizationState{}, &usagepb.CanvasState{
		Nodes: int32(len(expandedNodes)),
	}); err != nil {
		return nil, err
	}

	versioningEnabled, err := isCanvasVersioningEnabledForCanvas(canvas)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load canvas versioning: %v", err)
	}

	requestedVersionID := strings.TrimSpace(versionID)
	if requestedVersionID == "" {
		if versioningEnabled {
			return nil, status.Error(codes.FailedPrecondition, "canvas versioning is enabled for this canvas; version id is required")
		}

		return updateLiveCanvasWithoutVersioning(
			ctx,
			encryptor,
			registry,
			organizationUUID,
			canvas,
			nodes,
			edges,
			webhookBaseURL,
		)
	}

	if !versioningEnabled {
		return nil, status.Error(codes.FailedPrecondition, "canvas versioning is disabled for this canvas")
	}

	versionUUID, err := uuid.Parse(requestedVersionID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", err)
	}

	userUUID := uuid.MustParse(userID)
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, versionUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return err
		}

		if version.OwnerID == nil || *version.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}

		if version.IsPublished {
			return status.Error(codes.FailedPrecondition, "published versions are immutable")
		}

		if _, draftErr := models.FindCanvasDraftByVersionInTransaction(tx, canvasUUID, userUUID, version.ID); draftErr != nil {
			if errors.Is(draftErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.FailedPrecondition, "version is not your current edit version")
			}
			return draftErr
		}

		now := time.Now()
		version.Nodes = datatypes.NewJSONSlice(nodes)
		version.Edges = datatypes.NewJSONSlice(edges)
		version.UpdatedAt = &now

		if err := tx.Save(version).Error; err != nil {
			return err
		}

		return tx.Model(&models.CanvasUserDraft{}).
			Where("workflow_id = ? AND user_id = ? AND version_id = ?", canvasUUID, userUUID, version.ID).
			Update("updated_at", now).
			Error
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		log.WithError(err).Error("failed to update canvas version")
		return nil, status.Error(codes.Internal, "failed to update canvas version")
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.UpdateCanvasVersionResponse{
		Version: SerializeCanvasVersion(version, organizationID),
	}, nil
}

func updateLiveCanvasWithoutVersioning(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationUUID uuid.UUID,
	canvas *models.Canvas,
	nodes []models.Node,
	edges []models.Edge,
	webhookBaseURL string,
) (*pb.UpdateCanvasVersionResponse, error) {
	organizationID := organizationUUID.String()
	canvasID := canvas.ID
	var version *models.CanvasVersion

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasInTx, findCanvasErr := models.FindCanvasInTransaction(tx, organizationUUID, canvasID)
		if findCanvasErr != nil {
			return findCanvasErr
		}
		if canvasInTx.IsTemplate {
			return status.Error(codes.FailedPrecondition, "templates are read-only")
		}

		liveVersion, liveVersionErr := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvasInTx)
		if liveVersionErr != nil {
			if errors.Is(liveVersionErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.FailedPrecondition, "canvas live version not found")
			}
			return liveVersionErr
		}

		now := time.Now()
		existingNodesUnscoped, findNodesErr := models.FindCanvasNodesUnscopedInTransaction(tx, canvasID)
		if findNodesErr != nil {
			return findNodesErr
		}

		nodes, edges, _ = remapNodeIDsForConflicts(canvasID, nodes, edges, existingNodesUnscoped)

		existingNodes, findNodesErr := models.FindCanvasNodesInTransaction(tx, canvasID)
		if findNodesErr != nil {
			return findNodesErr
		}

		parentNodesByNodeID := make(map[string]*models.Node, len(nodes))
		for i := range nodes {
			parentNodesByNodeID[nodes[i].ID] = &nodes[i]
		}

		expandedNodes, expandErr := expandNodes(organizationID, nodes)
		if expandErr != nil {
			return expandErr
		}

		for _, node := range expandedNodes {
			if node.Type == models.NodeTypeWidget {
				continue
			}

			workflowNode, nodeLevelErrorMessage, upsertErr := upsertNode(tx, existingNodes, node, canvasID)
			if upsertErr != nil {
				return upsertErr
			}

			if nodeLevelErrorMessage != nil {
				errorNodeID := node.ID
				if workflowNode.ParentNodeID != nil {
					errorNodeID = *workflowNode.ParentNodeID
				}
				parentNode, ok := parentNodesByNodeID[errorNodeID]
				if ok {
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
					if ok {
						parentNode.ErrorMessage = &errorMsg
					}
				}
			}

			if workflowNode.ParentNodeID != nil {
				continue
			}

			parentNode, exists := parentNodesByNodeID[workflowNode.NodeID]
			if !exists {
				return status.Errorf(codes.Internal, "it was not possible to find the parent node %s", workflowNode.NodeID)
			}
			parentNode.Metadata = workflowNode.Metadata.Data()
		}

		if deleteErr := deleteNodes(tx, existingNodes, expandedNodes); deleteErr != nil {
			return deleteErr
		}

		liveVersion.Nodes = datatypes.NewJSONSlice(nodes)
		liveVersion.Edges = datatypes.NewJSONSlice(edges)
		liveVersion.UpdatedAt = &now
		if saveErr := tx.Save(liveVersion).Error; saveErr != nil {
			return saveErr
		}

		canvasInTx.UpdatedAt = &now
		canvasInTx.LiveVersionID = &liveVersion.ID
		if saveErr := tx.Save(canvasInTx).Error; saveErr != nil {
			return saveErr
		}

		version = liveVersion
		return nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		log.WithError(err).Error("failed to update live canvas")
		return nil, status.Error(codes.Internal, "failed to update live canvas")
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}
	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.UpdateCanvasVersionResponse{
		Version: SerializeCanvasVersion(version, organizationID),
	}, nil
}
