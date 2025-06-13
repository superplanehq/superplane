package stages

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListStages(ctx context.Context, req *pb.ListStagesRequest) (*pb.ListStagesResponse, error) {
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

	stages, err := canvas.ListStages()
	if err != nil {
		return nil, fmt.Errorf("failed to list stages for canvas: %w", err)
	}

	sources, err := canvas.ListEventSources()
	if err != nil {
		return nil, fmt.Errorf("failed to list event sources for canvas: %w", err)
	}

	serialized, err := serializeStages(stages, sources)
	if err != nil {
		return nil, err
	}

	response := &pb.ListStagesResponse{
		Stages: serialized,
	}

	return response, nil
}
