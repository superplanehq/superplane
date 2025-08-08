package eventsources

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeEventSource(ctx context.Context, canvasID string, idOrName string) (*pb.DescribeEventSourceResponse, error) {
	source, err := findEventSource(canvasID, idOrName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "event source not found")
		}

		log.Errorf("Error describing event source %s in canvas %s: %v", idOrName, canvasID, err)
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

func findEventSource(canvasID string, idOrName string) (*models.EventSource, error) {
	if idOrName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "must specify one of: id or name")
	}

	ID, err := uuid.Parse(idOrName)
	if err != nil {
		return models.FindExternalEventSourceByName(canvasID, idOrName)
	}

	return models.FindExternalEventSourceByID(canvasID, ID.String())
}
