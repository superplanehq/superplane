package actions

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/delivery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateCanvas(ctx context.Context, req *pb.CreateCanvasRequest) (*pb.CreateCanvasResponse, error) {
	orgID, err := uuid.Parse(req.OrganizationId)
	if err != nil {
		log.Errorf("Error reading organization id on %v for CreateCanvas: %v", req, err)
		return nil, err
	}

	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		log.Errorf("Error reading requester id on %v for CreateCanvas: %v", req, err)
		return nil, err
	}

	canvas, err := models.CreateCanvas(orgID, requesterID, req.Name)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating canvas on %v for CreateCanvas: %v", req, err)
		return nil, err
	}

	response := &pb.CreateCanvasResponse{
		Canvas: &pb.Canvas{
			Id:             canvas.ID.String(),
			Name:           canvas.Name,
			OrganizationId: canvas.OrganizationID.String(),
			CreatedAt:      timestamppb.New(*canvas.CreatedAt),
		},
	}

	return response, nil
}
