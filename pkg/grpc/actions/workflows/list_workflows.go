package workflows

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListWorkflows(ctx context.Context, registry *registry.Registry, organizationID string) (*pb.ListWorkflowsResponse, error) {
	workflows, err := models.ListWorkflows(organizationID)
	if err != nil {
		log.Errorf("failed to list workflows for organization %s: %v", organizationID, err)
		return nil, status.Error(codes.Internal, "failed to list workflows")
	}

	protoWorkflows := make([]*pb.Workflow, len(workflows))
	for i, workflow := range workflows {
		protoWorkflow, err := SerializeWorkflow(&workflow, false)
		if err != nil {
			return nil, err
		}

		protoWorkflows[i] = protoWorkflow
	}

	return &pb.ListWorkflowsResponse{
		Workflows: protoWorkflows,
	}, nil
}
