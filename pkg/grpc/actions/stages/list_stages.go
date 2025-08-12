package stages

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func ListStages(ctx context.Context, canvasID string) (*pb.ListStagesResponse, error) {
	stages, err := models.ListStages(canvasID)
	if err != nil {
		return nil, fmt.Errorf("failed to list stages for canvas: %w", err)
	}

	serialized, err := serializeStages(stages)
	if err != nil {
		return nil, err
	}

	response := &pb.ListStagesResponse{
		Stages: serialized,
	}

	return response, nil
}
