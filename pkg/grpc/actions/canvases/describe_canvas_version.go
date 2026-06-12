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

func DescribeCanvasVersion(ctx context.Context, organizationID string, canvasID string, versionID string) (*pb.DescribeCanvasVersionResponse, error) {
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

	version, err := models.FindCanvasVersion(canvas.ID, versionUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "version not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load version: %v", err)
	}

	userUUID := uuid.MustParse(userID)
	if version.State == models.CanvasVersionStatePublished {
		return &pb.DescribeCanvasVersionResponse{
			Version: SerializeCanvasVersionMetadata(version, organizationID),
		}, nil
	}

	canAccess := false
	if err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if models.IsUserOwnedDraftVersion(version, userUUID) && models.IsRegisteredDraftVersion(version) {
			canAccess = true
			return nil
		}

		if _, requestErr := models.FindCanvasChangeRequestByVersionInTransaction(tx, canvas.ID, version.ID); requestErr == nil {
			canAccess = true
			return nil
		} else if !errors.Is(requestErr, gorm.ErrRecordNotFound) {
			return requestErr
		}
		return nil
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve version access: %v", err)
	}

	if !canAccess {
		return nil, status.Error(codes.PermissionDenied, "version is not visible in current flow")
	}

	state, _, err := stagingSummaryForVersion(version.ID)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeCanvasVersionResponse{
		Version:      SerializeCanvasVersionMetadata(version, organizationID),
		StagingSummary: state,
	}, nil
}
