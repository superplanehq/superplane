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

func DiscardCanvasDraft(
	ctx context.Context,
	organizationID string,
	canvasID string,
) (*pb.DiscardCanvasDraftResponse, error) {
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

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		draft, findErr := models.FindCanvasDraftInTransaction(tx, canvasUUID, userUUID)
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "no draft found for this user")
			}
			return findErr
		}

		if deleteVersionErr := tx.Delete(&models.CanvasVersion{}, "id = ?", draft.VersionID).Error; deleteVersionErr != nil {
			return deleteVersionErr
		}

		return tx.Delete(&models.CanvasUserDraft{}, "workflow_id = ? AND user_id = ?", canvasUUID, userUUID).Error
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to discard draft: %v", err)
	}

	return &pb.DiscardCanvasDraftResponse{}, nil
}
