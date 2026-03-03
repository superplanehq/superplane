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
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ResolveCanvasChangeRequest(
	ctx context.Context,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	changeRequestID string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
) (*pb.ResolveCanvasChangeRequestResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	changeRequestUUID, err := uuid.Parse(changeRequestID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid change request id: %v", err)
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

	nodes, edges, err := ParseCanvas(registry, organizationID, pbCanvas)
	if err != nil {
		return nil, err
	}

	nodes, edges, err = applyCanvasAutoLayout(nodes, edges, autoLayout, registry)
	if err != nil {
		return nil, err
	}

	userUUID := uuid.MustParse(userID)
	var version *models.CanvasVersion
	var request *models.CanvasChangeRequest

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasInTx, findCanvasErr := models.FindCanvasInTransaction(tx, organizationUUID, canvasUUID)
		if findCanvasErr != nil {
			return findCanvasErr
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

		if request.OwnerID == nil || *request.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "change request owner mismatch")
		}

		version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, request.VersionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "change request version not found")
			}
			return err
		}

		if version.IsPublished {
			return status.Error(codes.FailedPrecondition, "published versions are immutable")
		}

		if version.OwnerID == nil || *version.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}

		now := time.Now()
		version.Nodes = datatypes.NewJSONSlice(nodes)
		version.Edges = datatypes.NewJSONSlice(edges)
		version.UpdatedAt = &now
		if canvasInTx.LiveVersionID != nil {
			liveVersionID := *canvasInTx.LiveVersionID
			version.BasedOnVersionID = &liveVersionID
		} else {
			version.BasedOnVersionID = nil
		}

		if saveErr := tx.Save(version).Error; saveErr != nil {
			return saveErr
		}

		draft := models.CanvasUserDraft{
			WorkflowID: canvasUUID,
			UserID:     userUUID,
			VersionID:  version.ID,
			CreatedAt:  &now,
			UpdatedAt:  &now,
		}

		if saveErr := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "workflow_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"version_id": version.ID,
				"updated_at": now,
			}),
		}).Create(&draft).Error; saveErr != nil {
			return saveErr
		}

		return refreshCanvasChangeRequestDiffInTransaction(tx, canvasInTx, version, request)
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to resolve canvas change request: %v", err)
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.ResolveCanvasChangeRequestResponse{
		Version:       SerializeCanvasVersion(version, organizationID),
		ChangeRequest: SerializeCanvasChangeRequest(request, version, organizationID),
	}, nil
}
