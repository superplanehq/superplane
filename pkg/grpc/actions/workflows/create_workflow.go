package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
)

func CreateWorkflow(ctx context.Context, registry *registry.Registry, organizationID string, workflow *pb.Workflow) (*pb.CreateWorkflowResponse, error) {
	nodes, edges, err := ParseWorkflow(registry, workflow)
	if err != nil {
		return nil, err
	}

	orgID, _ := uuid.Parse(organizationID)
	now := time.Now()

	model := &models.Workflow{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           workflow.Name,
		Description:    workflow.Description,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Nodes:          datatypes.NewJSONSlice(nodes),
		Edges:          datatypes.NewJSONSlice(edges),
	}

	if err := database.Conn().Create(model).Error; err != nil {
		return nil, err
	}

	return &pb.CreateWorkflowResponse{
		Workflow: SerializeWorkflow(model),
	}, nil
}
