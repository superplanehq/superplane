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

func CreateCanvasVersion(ctx context.Context, organizationID string, canvasID string) (*pb.CreateCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orgUUID := uuid.MustParse(organizationID)
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	userUUID := uuid.MustParse(userID)
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		draft, draftErr := models.FindCanvasDraftInTransaction(tx, canvasUUID, userUUID)
		if draftErr == nil {
			version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, draft.VersionID)
			return err
		}

		if !errors.Is(draftErr, gorm.ErrRecordNotFound) {
			return draftErr
		}

		version, err = models.SaveCanvasDraftInTransaction(
			tx,
			canvas.ID,
			userUUID,
			canvas.LiveVersionID,
			canvas.Nodes,
			canvas.Edges,
		)

		return err
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create canvas version: %v", err)
	}

	return &pb.CreateCanvasVersionResponse{
		Version: SerializeCanvasVersion(version, organizationID),
	}, nil
}
