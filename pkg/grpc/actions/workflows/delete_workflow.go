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

func DeleteWorkflow(ctx context.Context, registry *registry.Registry, organizationID uuid.UUID, id string) (*pb.DeleteWorkflowResponse, error) {
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow id: %v", err)
	}

	workflow, err := models.FindWorkflow(organizationID, workflowID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	// Perform soft delete on the workflow with name suffix
	// The cleanup worker will handle the actual deletion of nodes and related data
	err = workflow.SoftDelete()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete workflow: %v", err)
	}

	return &pb.DeleteWorkflowResponse{}, nil
}
