package workflows

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListWorkflows(ctx context.Context, registry *registry.Registry, organizationID string) (*pb.ListWorkflowsResponse, error) {
	workflows, err := models.ListWorkflows(organizationID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoWorkflows := make([]*pb.Workflow, len(workflows))
	for i, workflow := range workflows {
		protoWorkflows[i] = SerializeWorkflow(&workflow, false)
	}

	return &pb.ListWorkflowsResponse{
		Workflows: protoWorkflows,
	}, nil
}
