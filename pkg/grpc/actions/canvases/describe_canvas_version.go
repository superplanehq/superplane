package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
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
	isOwnedByUser := version.OwnerID != nil && *version.OwnerID == userUUID
	isLiveVersion := canvas.LiveVersionID != nil && *canvas.LiveVersionID == version.ID

	if version.IsPublished {
		if !isLiveVersion && !isOwnedByUser {
			return nil, status.Error(codes.PermissionDenied, "version owner mismatch")
		}
	} else if !isOwnedByUser {
		return nil, status.Error(codes.PermissionDenied, "version owner mismatch")
	}

	return &pb.DescribeCanvasVersionResponse{
		Version: SerializeCanvasVersion(version, organizationID),
	}, nil
}
