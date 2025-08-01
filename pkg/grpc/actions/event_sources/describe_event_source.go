package eventsources

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeEventSource(ctx context.Context, canvasID string, req *pb.DescribeEventSourceRequest) (*pb.DescribeEventSourceResponse, error) {
	canvas, err := models.FindCanvasByID(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	logger := logging.ForCanvas(canvas)
	source, err := findEventSource(canvas, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "event source not found")
		}

		logger.Errorf("Error describing event source. Request: %v. Error: %v", req, err)
		return nil, err
	}

	protoSource, err := serializeEventSource(*source)
	if err != nil {
		return nil, err
	}

	response := &pb.DescribeEventSourceResponse{
		EventSource: protoSource,
	}

	return response, nil
}

func findEventSource(canvas *models.Canvas, req *pb.DescribeEventSourceRequest) (*models.EventSource, error) {
	if req.Name == "" && req.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "must specify one of: id or name")
	}

	if req.Name != "" {
		return canvas.FindEventSourceByName(req.Name)
	}

	ID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ID")
	}

	return canvas.FindEventSourceByID(ID)
}
