package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func DescribeCanvasVersion(ctx context.Context, organizationID string, canvasID string, versionID string) (*pb.DescribeCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid version id")
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasUUID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	version, err := models.FindCanvasVersion(canvas.ID, versionUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "version not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load version")
	}

	userUUID := uuid.MustParse(userID)
	if version.State == models.CanvasVersionStatePublished {
		return &pb.DescribeCanvasVersionResponse{
			Version: SerializeCanvasVersionMetadata(version, organizationID, nil),
		}, nil
	}

	canAccess := false
	if models.IsUserOwnedDraftVersion(version, userUUID) && models.IsRegisteredDraftVersion(version) {
		canAccess = true
	}

	if !canAccess {
		return nil, grpcerrors.PermissionDenied(nil, "version is not visible in current flow")
	}

	state, _, err := stagingSummaryForVersion(version.ID)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeCanvasVersionResponse{
		Version:        SerializeCanvasVersionMetadata(version, organizationID, nil),
		StagingSummary: state,
	}, nil
}
