package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/delivery"
)

func ListEventSources(ctx context.Context, req *pb.ListEventSourcesRequest) (*pb.ListEventSourcesResponse, error) {
	err := ValidateUUIDs(req.OrganizationId, req.CanvasId)
	if err != nil {
		return nil, err
	}

	canvas, err := models.FindCanvasByID(req.CanvasId, req.OrganizationId)
	if err != nil {
		return nil, err
	}

	sources, err := canvas.ListEventSources()
	if err != nil {
		return nil, err
	}

	response := &pb.ListEventSourcesResponse{
		EventSources: serializeEventSources(sources),
	}

	return response, nil
}

func serializeEventSources(eventSources []models.EventSource) []*pb.EventSource {
	sources := []*pb.EventSource{}
	for _, source := range eventSources {
		sources = append(sources, serializeEventSource(source))
	}

	return sources
}
