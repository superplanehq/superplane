package actions

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/delivery"
)

func ListStages(ctx context.Context, req *pb.ListStagesRequest) (*pb.ListStagesResponse, error) {
	err := ValidateUUIDs(req.OrganizationId, req.CanvasId)
	if err != nil {
		return nil, err
	}

	canvas, err := models.FindCanvasByID(req.CanvasId, req.OrganizationId)
	if err != nil {
		return nil, err
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
