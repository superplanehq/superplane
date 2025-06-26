package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateCanvas(ctx context.Context, req *pb.CreateCanvasRequest, authorizationService authorization.Authorization) (*pb.CreateCanvasResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)

	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if req.Canvas == nil || req.Canvas.Metadata == nil || req.Canvas.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "canvas name is required")
	}

	orgID, err := uuid.Parse(req.OrganizationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	_, err = models.FindOrganizationByID(orgID.String())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "organization not found")
	}

	userIDUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	canvas, err := models.CreateCanvas(userIDUUID, orgID, req.Canvas.Metadata.Name)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating canvas on %v for CreateCanvas: %v", req, err)
		return nil, err
	}

	response := &pb.CreateCanvasResponse{
		Canvas: &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{
				Id:        canvas.ID.String(),
				Name:      canvas.Name,
				CreatedAt: timestamppb.New(*canvas.CreatedAt),
			},
		},
	}

	err = authorizationService.SetupCanvasRoles(canvas.ID.String())

	if err != nil {
		log.Errorf("Error setting up canvas roles on %v for CreateCanvas: %v", req, err)
		return nil, err
	}

	err = authorizationService.AssignRole(userID, authorization.RoleCanvasOwner, canvas.ID.String(), authorization.DomainCanvas)
	if err != nil {
		log.Errorf("Error assigning canvas owner role on %v for CreateCanvas: %v", req, err)
		return nil, err
	}

	return response, nil
}
