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

func PublishCanvasVersion(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	expectedLiveVersionID string,
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

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", err)
	}

	var expectedLiveVersionUUID *uuid.UUID
	if expectedLiveVersionID != "" {
		parsedExpected, err := uuid.Parse(expectedLiveVersionID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid expected live version id: %v", err)
		}
		expectedLiveVersionUUID = &parsedExpected
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	userUUID := uuid.MustParse(userID)
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasForUpdate, err := models.FindCanvasInTransaction(tx, uuid.MustParse(organizationID), canvasUUID)
		if err != nil {
			return err
		}

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

		if expectedLiveVersionUUID != nil {
			if canvasForUpdate.LiveVersionID == nil || *canvasForUpdate.LiveVersionID != *expectedLiveVersionUUID {
				return status.Error(codes.FailedPrecondition, "live version changed")
			}
		}

		if version.BasedOnVersionID != nil {
			if canvasForUpdate.LiveVersionID == nil || *canvasForUpdate.LiveVersionID != *version.BasedOnVersionID {
				return status.Error(codes.FailedPrecondition, "version was created from an outdated live version")
			}
		}

		nodes := append([]models.Node(nil), version.Nodes...)
		edges := append([]models.Edge(nil), version.Edges...)

		existingNodesUnscoped, err := models.FindCanvasNodesUnscopedInTransaction(tx, canvasUUID)
		if err != nil {
			return err
		}

		nodes, edges, _ = remapNodeIDsForConflicts(nodes, edges, existingNodesUnscoped)

		parentNodesByNodeID := make(map[string]*models.Node)
		for i := range nodes {
			parentNodesByNodeID[nodes[i].ID] = &nodes[i]
		}

		expandedNodes, err := expandNodes(organizationID, nodes)
		if err != nil {
			return err
		}

		now := time.Now()

		existingNodes, err := models.FindCanvasNodesInTransaction(tx, canvasUUID)
		if err != nil {
			return err
		}

		for _, node := range expandedNodes {
			if node.Type == models.NodeTypeWidget {
				continue
			}

			workflowNode, err := upsertNode(tx, existingNodes, node, canvasUUID)
			if err != nil {
				return err
			}

			if workflowNode.State != models.CanvasNodeStateReady {
				continue
			}

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

			if workflowNode.ParentNodeID != nil {
				continue
			}

			parentNode, ok := parentNodesByNodeID[workflowNode.NodeID]
			if !ok {
				log.Errorf("Parent node %s not found", workflowNode.NodeID)
				return status.Errorf(codes.Internal, "It was not possible to find the parent node %s", workflowNode.NodeID)
			}
			parentNode.Metadata = workflowNode.Metadata.Data()
		}

		canvasForUpdate.UpdatedAt = &now
		canvasForUpdate.Nodes = datatypes.NewJSONSlice(nodes)
		canvasForUpdate.Edges = datatypes.NewJSONSlice(edges)
		canvasForUpdate.LiveVersionID = &version.ID
		if err := tx.Save(canvasForUpdate).Error; err != nil {
			return err
		}

		if err := deleteNodes(tx, existingNodes, expandedNodes); err != nil {
			return err
		}

		version.Nodes = datatypes.NewJSONSlice(nodes)
		version.Edges = datatypes.NewJSONSlice(edges)
		version.IsPublished = true
		version.PublishedAt = &now
		version.UpdatedAt = &now
		if err := tx.Save(version).Error; err != nil {
			return err
		}

		if err := tx.Delete(&models.CanvasUserDraft{}, "workflow_id = ? AND user_id = ?", canvasUUID, userUUID).Error; err != nil {
			return err
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

	return &pb.PublishCanvasVersionResponse{
		Canvas:  protoCanvas,
		Version: SerializeCanvasVersion(version, organizationID),
	}, nil
}
