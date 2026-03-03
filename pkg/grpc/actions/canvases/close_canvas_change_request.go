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

func CloseCanvasChangeRequest(
	ctx context.Context,
	organizationID string,
	canvasID string,
	changeRequestID string,
) (*pb.CloseCanvasChangeRequestResponse, error) {
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

	userUUID := uuid.MustParse(userID)
	var request *models.CanvasChangeRequest
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		_, findCanvasErr := models.FindCanvasInTransaction(tx, organizationUUID, canvasUUID)
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

		if request.Status == models.CanvasChangeRequestStatusClosed {
			return nil
		}

		now := time.Now()
		request.Status = models.CanvasChangeRequestStatusClosed
		request.UpdatedAt = &now
		return tx.Save(request).Error
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to close canvas change request: %v", err)
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.CloseCanvasChangeRequestResponse{
		ChangeRequest: SerializeCanvasChangeRequest(request, version, organizationID),
	}, nil
}
