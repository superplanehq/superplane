package eventsources

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListEventSources(ctx context.Context, req *pb.ListEventSourcesRequest) (*pb.ListEventSourcesResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasIdOrName)

	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "canvas not found")
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
