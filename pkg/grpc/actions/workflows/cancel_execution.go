package workflows

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CancelExecution(ctx context.Context, registry *registry.Registry, workflowID, executionID uuid.UUID) (*pb.CancelExecutionResponse, error) {
	execution, err := models.FindNodeExecution(workflowID, executionID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "execution not found")
	}

	err = execution.Cancel()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CancelExecutionResponse{}, nil
}
