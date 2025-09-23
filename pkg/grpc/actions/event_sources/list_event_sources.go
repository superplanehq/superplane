package eventsources

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListEventSources(ctx context.Context, canvasID string) (*pb.ListEventSourcesResponse, error) {
	sources, err := models.ListEventSources(canvasID)
	if err != nil {
		log.Errorf("error listing event sources for canvas %s: %v", canvasID, err)
		return nil, status.Error(codes.Internal, "error listing event sources")
	}

	protoSources, err := serializeEventSources(sources)
	if err != nil {
		return nil, err
	}

	response := &pb.ListEventSourcesResponse{
		EventSources: protoSources,
	}

	return response, nil
}

func serializeEventSources(eventSources []models.EventSource) ([]*pb.EventSource, error) {
	if len(eventSources) == 0 {
		return []*pb.EventSource{}, nil
	}

	lastEvents, err := models.GetEventSourcesLastEvents(eventSources)
	if err != nil {
		return nil, fmt.Errorf("failed to get event source last events: %w", err)
	}

	sources := []*pb.EventSource{}
	for _, source := range eventSources {
		var lastEvent *models.Event
		if event, exists := lastEvents[source.ID]; exists {
			lastEvent = event
		}

		protoSource, err := serializeEventSource(source, lastEvent)
		if err != nil {
			return nil, err
		}

		sources = append(sources, protoSource)
	}

	return sources, nil
}
