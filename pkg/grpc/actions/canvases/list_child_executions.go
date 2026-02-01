package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListChildExecutions(ctx context.Context, registry *registry.Registry, workflowID, executionID uuid.UUID) (*pb.ListChildExecutionsResponse, error) {
	executions, err := models.FindChildExecutions(executionID, []string{
		models.CanvasNodeExecutionStatePending,
		models.CanvasNodeExecutionStateStarted,
		models.CanvasNodeExecutionStateFinished,
	})

	if err != nil {
		return nil, err
	}

	serialized, err := SerializeNodeExecutions(executions, []models.CanvasNodeExecution{})
	if err != nil {
		return nil, err
	}

	return &pb.ListChildExecutionsResponse{
		Executions: serialized,
	}, nil
}
