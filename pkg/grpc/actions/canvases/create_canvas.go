package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateCanvas(ctx context.Context, authService authorization.Authorization, orgID string, canvas *pb.Canvas) (*pb.CreateCanvasResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if canvas == nil || canvas.Metadata == nil || canvas.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "canvas name is required")
	}

	// TODO: we should use transaction here

	newCanvas, err := models.CreateCanvas(
		uuid.MustParse(userID),
		uuid.MustParse(orgID),
		canvas.Metadata.Name,
		canvas.Metadata.Description,
	)

	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating canvas: %v", err)
		return nil, err
	}

	err = authService.SetupCanvasRoles(newCanvas.ID.String())
	if err != nil {
		log.Errorf("Error setting up roles for canvas %s: %v", newCanvas.ID.String(), err)
		return nil, err
	}

	err = authService.AssignRole(userID, models.RoleCanvasOwner, newCanvas.ID.String(), models.DomainTypeCanvas)
	if err != nil {
		log.Errorf("Error assigning owner role for canvas %s: %v", newCanvas.ID.String(), err)
		return nil, err
	}

	return &pb.CreateCanvasResponse{
		Canvas: &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{
				Id:          newCanvas.ID.String(),
				Name:        newCanvas.Name,
				Description: newCanvas.Description,
				CreatedAt:   timestamppb.New(*newCanvas.CreatedAt),
			},
		},
	}, nil
}
