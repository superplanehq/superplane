package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DiscardCanvasVersion(ctx context.Context, organizationID string, canvasID string, versionID string) (*pb.DiscardCanvasVersionResponse, error) {
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

	userUUID := uuid.MustParse(userID)
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		version, err := models.FindCanvasVersionInTransaction(tx, canvasUUID, versionUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return err
		}

		if version.IsPublished {
			return status.Error(codes.FailedPrecondition, "published versions are immutable")
		}

		if version.OwnerID == nil || *version.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}

		if err := tx.
			Where("workflow_id = ? AND user_id = ? AND version_id = ?", canvasUUID, userUUID, versionUUID).
			Delete(&models.CanvasUserDraft{}).
			Error; err != nil {
			return err
		}

		return tx.
			Where("workflow_id = ? AND id = ?", canvasUUID, versionUUID).
			Delete(&models.CanvasVersion{}).
			Error
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to discard canvas version: %v", err)
	}

	return &pb.DiscardCanvasVersionResponse{}, nil
}
