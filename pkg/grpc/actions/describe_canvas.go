package actions

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/delivery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func DescribeCanvas(ctx context.Context, req *pb.DescribeCanvasRequest) (*pb.DescribeCanvasResponse, error) {
	err := ValidateUUIDs(req.OrganizationId, req.Id)
	if err != nil {
		return nil, err
	}

	canvas, err := models.FindCanvasByID(req.Id, req.OrganizationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}

		log.Errorf("Error describing canvas. Request: %v. Error: %v", req, err)
		return nil, err
	}

	response := &pb.DescribeCanvasResponse{
		Canvas: &pb.Canvas{
			Id:             canvas.ID.String(),
			Name:           canvas.Name,
			OrganizationId: canvas.OrganizationID.String(),
			CreatedAt:      timestamppb.New(*canvas.CreatedAt),
			CreatedBy:      canvas.CreatedBy.String(),
		},
	}

	return response, nil
}
