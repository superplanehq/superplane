package canvases

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListCanvases(ctx context.Context, orgID string, authorizationService authorization.Authorization) (*pb.ListCanvasesResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)

	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	roles, err := authorizationService.GetUserRolesForCanvasWithOrgContext(userID, "*", orgID)
	if err != nil {
		return nil, err
	}

	//
	// If user has global canvas role, he can see all canvases in the organization
	//
	var canvases []models.Canvas
	if len(roles) > 0 {
		canvases, err = models.ListCanvasesByOrgID(orgID)
	} else {
		accessibleCanvasIDs, err := authorizationService.GetAccessibleCanvasesForUser(userID)
		if err != nil {
			log.Errorf("failed to list canvases IDs by org ID: %v", err)
			return nil, status.Error(codes.Internal, "failed to list canvases IDs")
		}
		canvases, err = models.ListCanvasesByIDs(accessibleCanvasIDs, orgID)
	}

	if err != nil {
		log.Errorf("failed to list canvases IDs by org ID: %v", err)
		return nil, status.Error(codes.Internal, "failed to list canvases IDs")
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
				Id:          canvas.ID.String(),
				Name:        canvas.Name,
				Description: canvas.Description,
				CreatedBy:   canvas.CreatedBy.String(),
				CreatedAt:   timestamppb.New(*canvas.CreatedAt),
			},
		})
	}

	return out
}
