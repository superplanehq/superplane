package actions

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func DescribeCanvas(ctx context.Context, req *pb.DescribeCanvasRequest) (*pb.DescribeCanvasResponse, error) {
	err := ValidateUUIDs(req.Id)
	if err != nil {
		return nil, err
	}

	var canvas *models.Canvas
	if req.Name != "" {
		canvas, err = models.FindCanvasByName(req.Name)

	} else {
		canvas, err = models.FindCanvasByID(req.Id)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}

		log.Errorf("Error describing canvas. Request: %v. Error: %v", req, err)
		return nil, err
	}

	response := &pb.DescribeCanvasResponse{
		Canvas: &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{
				Id:        canvas.ID.String(),
				Name:      canvas.Name,
				CreatedAt: timestamppb.New(*canvas.CreatedAt),
				CreatedBy: canvas.CreatedBy.String(),
			},
		},
	}

	return response, nil
}
