package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func DescribeCanvasVersion(ctx context.Context, db *gorm.DB, canvas *models.Canvas, versionID string) (*pb.DescribeCanvasVersionResponse, error) {
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid version id")
	}

	version, err := models.FindCanvasVersionInTransaction(db, canvas.ID, versionUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "version not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load version")
	}

	return &pb.DescribeCanvasVersionResponse{
		Version: SerializeCanvasVersion(version, canvas.OrganizationID.String(), nil),
	}, nil
}
