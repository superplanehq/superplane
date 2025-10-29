package workflows

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListChildExecutions(ctx context.Context, registry *registry.Registry, workflowID, executionID uuid.UUID) (*pb.ListChildExecutionsResponse, error) {
	executions, err := models.FindChildExecutions(executionID, []string{
		models.WorkflowNodeExecutionStatePending,
		models.WorkflowNodeExecutionStateStarted,
		models.WorkflowNodeExecutionStateFinished,
	})

	if err != nil {
		return nil, err
	}

	serialized, err := SerializeNodeExecutions(executions, []models.WorkflowNodeExecution{})
	if err != nil {
		return nil, err
	}

	return &pb.ListChildExecutionsResponse{
		Executions: serialized,
	}, nil
}
