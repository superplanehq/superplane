package workflows

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ResolveExecutionErrors(ctx context.Context, workflowID uuid.UUID, executionIDs []uuid.UUID) (*pb.ResolveExecutionErrorsResponse, error) {
	_ = ctx
	uniqueExecutionIDs := uniqueExecutionIDs(executionIDs)
	if len(uniqueExecutionIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "execution_ids are required")
	}

	executions, err := models.FindNodeExecutionsByIDs(workflowID, uniqueExecutionIDs)
	if err != nil {
		return nil, err
	}

	if len(executions) != len(uniqueExecutionIDs) {
		return nil, status.Error(codes.NotFound, "execution not found")
	}

	invalidIDs := invalidErrorResolutionIDs(executions)
	if len(invalidIDs) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "executions not in error state: %s", strings.Join(invalidIDs, ", "))
	}

	if err := models.ResolveExecutionErrors(workflowID, uniqueExecutionIDs); err != nil {
		return nil, err
	}

	return &pb.ResolveExecutionErrorsResponse{}, nil
}

func uniqueExecutionIDs(executionIDs []uuid.UUID) []uuid.UUID {
	unique := make([]uuid.UUID, 0, len(executionIDs))
	seen := make(map[uuid.UUID]struct{}, len(executionIDs))
	for _, id := range executionIDs {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func invalidErrorResolutionIDs(executions []models.WorkflowNodeExecution) []string {
	invalidIDs := make([]string, 0)
	for _, execution := range executions {
		if execution.ResultReason == models.WorkflowNodeExecutionResultReasonError ||
			execution.ResultReason == models.WorkflowNodeExecutionResultReasonErrorResolved {
			continue
		}
		invalidIDs = append(invalidIDs, execution.ID.String())
	}
	return invalidIDs
}
