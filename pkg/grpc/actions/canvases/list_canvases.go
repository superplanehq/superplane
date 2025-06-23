package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListCanvases(ctx context.Context, req *pb.ListCanvasesRequest, authorizationService authorization.Authorization) (*pb.ListCanvasesResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)

	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	accessibleCanvasIDs, err := authorizationService.GetAccessibleCanvasesForUser(userID)
	if err != nil {
		return nil, err
	}

	canvases, err := models.ListCanvasesByIDs(accessibleCanvasIDs, req.OrganizationId)
	if err != nil {
		return nil, err
	}

	response := &pb.ListCanvasesResponse{
		Canvases: serializeCanvases(canvases),
	}

	return response, nil
}

func serializeCanvases(in []models.Canvas) []*pb.Canvas {
	out := []*pb.Canvas{}
	for _, canvas := range in {
		out = append(out, &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{
				Id:        canvas.ID.String(),
				Name:      canvas.Name,
				CreatedBy: canvas.CreatedBy.String(),
				CreatedAt: timestamppb.New(*canvas.CreatedAt),
			},
		})
	}

	return out
}
