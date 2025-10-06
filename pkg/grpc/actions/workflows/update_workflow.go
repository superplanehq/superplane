package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func UpdateWorkflow(ctx context.Context, registry *registry.Registry, organizationID string, id string, workflow *pb.Workflow) (*pb.UpdateWorkflowResponse, error) {
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow id: %v", err)
	}

	nodes, edges, err := ParseWorkflow(registry, workflow)
	if err != nil {
		return nil, err
	}

	var existing models.Workflow
	if err := database.Conn().Where("id = ? AND organization_id = ?", workflowID, organizationID).First(&existing).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	now := time.Now()
	existing.Name = workflow.Name
	existing.Description = workflow.Description
	existing.UpdatedAt = &now
	existing.Nodes = datatypes.NewJSONSlice(nodes)
	existing.Edges = datatypes.NewJSONSlice(edges)

	if err := database.Conn().Save(&existing).Error; err != nil {
		return nil, err
	}

	return &pb.UpdateWorkflowResponse{
		Workflow: SerializeWorkflow(&existing),
	}, nil
}
