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

	statusInfo, err := models.GetEventSourcesStatusInfo(eventSources)
	if err != nil {
		return nil, fmt.Errorf("failed to get event source status info: %w", err)
	}

	sources := []*pb.EventSource{}
	for _, source := range eventSources {
		var sourceStatus *models.EventSourceStatusInfo
		if info, exists := statusInfo[source.ID]; exists {
			sourceStatus = info
		}

		protoSource, err := serializeEventSource(source, sourceStatus)
		if err != nil {
			return nil, err
		}

		sources = append(sources, protoSource)
	}

	return sources, nil
}
