package eventsources

import (
	"context"
	"errors"
	"fmt"

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

	lastEvents, err := models.GetEventSourcesLastEvents([]models.EventSource{*source})
	if err != nil {
		return nil, fmt.Errorf("failed to get event source last events: %w", err)
	}

	var lastEvent *models.Event
	if event, exists := lastEvents[source.ID]; exists {
		lastEvent = event
	}

	protoSource, err := serializeEventSource(*source, lastEvent)
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
		return nil, status.Errorf(codes.InvalidArgument, "must specify either the ID or name of the stage")
	}

	ID, err := uuid.Parse(idOrName)
	if err != nil {
		return models.FindExternalEventSourceByName(canvasID, idOrName)
	}

	return models.FindExternalEventSourceByID(canvasID, ID.String())
}
