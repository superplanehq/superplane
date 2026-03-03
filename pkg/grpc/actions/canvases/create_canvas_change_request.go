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
	"gorm.io/gorm"
)

func CreateCanvasChangeRequest(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
) (*pb.CreateCanvasChangeRequestResponse, error) {
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

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasUUID)
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
	var request *models.CanvasChangeRequest
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasInTx, findCanvasErr := models.FindCanvasInTransaction(tx, uuid.MustParse(organizationID), canvasUUID)
		if findCanvasErr != nil {
			return findCanvasErr
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

		if version.IsPublished {
			return status.Error(codes.FailedPrecondition, "published versions cannot create change requests")
		}

		existingRequest, findErr := models.FindCanvasChangeRequestByVersionInTransaction(tx, canvasUUID, versionUUID)
		if findErr == nil {
			request = existingRequest
			if request.Status == models.CanvasChangeRequestStatusClosed {
				request.Status = models.CanvasChangeRequestStatusOpen
			}
			return refreshCanvasChangeRequestDiffInTransaction(tx, canvasInTx, version, request)
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		now := time.Now()
		request = &models.CanvasChangeRequest{
			ID:               uuid.New(),
			WorkflowID:       canvasUUID,
			VersionID:        versionUUID,
			OwnerID:          &userUUID,
			BasedOnVersionID: version.BasedOnVersionID,
			Status:           models.CanvasChangeRequestStatusOpen,
			CreatedAt:        &now,
			UpdatedAt:        &now,
		}

		if createErr := tx.Create(request).Error; createErr != nil {
			return createErr
		}

		return refreshCanvasChangeRequestDiffInTransaction(tx, canvasInTx, version, request)
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to create canvas change request: %v", err)
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.CreateCanvasChangeRequestResponse{
		ChangeRequest: SerializeCanvasChangeRequest(request, version, organizationID),
	}, nil
}
