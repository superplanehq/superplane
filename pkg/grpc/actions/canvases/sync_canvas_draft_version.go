package canvases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SyncCanvasDraftVersion(
	ctx context.Context,
	organizationID string,
	canvasID string,
) (*pb.SyncCanvasDraftVersionResponse, error) {
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

	sandboxModeEnabled, modeErr := isCanvasSandboxModeEnabled(organizationID)
	if modeErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to load organization sandbox mode: %v", modeErr)
	}
	if sandboxModeEnabled {
		return nil, status.Error(codes.FailedPrecondition, "canvas versioning is disabled in sandbox mode")
	}

	userUUID := uuid.MustParse(userID)
	var version *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasInTx, findCanvasErr := models.FindCanvasInTransaction(tx, organizationUUID, canvasUUID)
		if findCanvasErr != nil {
			return findCanvasErr
		}
		if canvasInTx.LiveVersionID == nil {
			return status.Error(codes.FailedPrecondition, "canvas live version not found")
		}

		draft, draftErr := models.FindCanvasDraftInTransaction(tx, canvasUUID, userUUID)
		if draftErr != nil {
			if errors.Is(draftErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.FailedPrecondition, "no edit version found for this user")
			}
			return draftErr
		}

		version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, draft.VersionID)
		if err != nil {
			return err
		}
		if version.OwnerID == nil || *version.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}
		if version.IsPublished {
			return status.Error(codes.FailedPrecondition, "published versions are immutable")
		}

		baseNodes, baseEdges, liveNodes, liveEdges, resolveErr := resolveCanvasVersionBaseAndLiveInTransaction(
			tx,
			canvasInTx,
			version,
		)
		if resolveErr != nil {
			return resolveErr
		}

		changedSet := resolveChangedNodeIDSet(baseNodes, baseEdges, version.Nodes, version.Edges)
		changedNodeIDs := resolveOrderedNodeIDs(changedSet, version.Nodes, liveNodes, baseNodes)
		rebasedNodes, rebasedEdges := mergeCanvasVersionIntoLive(
			baseNodes,
			baseEdges,
			liveNodes,
			liveEdges,
			version.Nodes,
			version.Edges,
			changedNodeIDs,
		)

		now := time.Now()
		liveVersionID := *canvasInTx.LiveVersionID
		version.BasedOnVersionID = &liveVersionID
		version.Nodes = datatypes.NewJSONSlice(rebasedNodes)
		version.Edges = datatypes.NewJSONSlice(rebasedEdges)
		version.UpdatedAt = &now
		if saveErr := tx.Save(version).Error; saveErr != nil {
			return saveErr
		}

		draft.UpdatedAt = &now
		return tx.Save(draft).Error
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to sync canvas draft version: %v", err)
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.SyncCanvasDraftVersionResponse{
		Version: SerializeCanvasVersion(version, organizationID),
	}, nil
}
