package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListCanvasVersions(ctx context.Context, organizationID string, canvasID string) (*pb.ListCanvasVersionsResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	userUUID := uuid.MustParse(userID)
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	versions, err := models.ListCanvasVersions(canvas.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list canvas versions: %v", err)
	}

	protoVersions := make([]*pb.CanvasVersion, 0, len(versions))
	for i := range versions {
		version := versions[i]
		if !version.IsPublished && (version.OwnerID == nil || *version.OwnerID != userUUID) {
			continue
		}

		protoVersions = append(protoVersions, SerializeCanvasVersion(&versions[i], organizationID))
	}

	return &pb.ListCanvasVersionsResponse{
		Versions: protoVersions,
	}, nil
}
