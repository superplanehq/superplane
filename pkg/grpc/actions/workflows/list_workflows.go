package workflows

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListWorkflows(ctx context.Context, registry *registry.Registry, organizationID string) (*pb.ListWorkflowsResponse, error) {
	var workflows []models.Workflow

	if err := database.Conn().Where("organization_id = ?", organizationID).Find(&workflows).Error; err != nil {
		return nil, err
	}

	protoWorkflows := make([]*pb.Workflow, len(workflows))
	for i, workflow := range workflows {
		protoWorkflows[i] = SerializeWorkflow(&workflow)
	}

	return &pb.ListWorkflowsResponse{
		Workflows: protoWorkflows,
	}, nil
}
