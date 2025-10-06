package blueprints

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/blueprints"
	"github.com/superplanehq/superplane/pkg/registry"
)

func CreateBlueprint(ctx context.Context, registry *registry.Registry, organizationID string, blueprint *pb.Blueprint) (*pb.CreateBlueprintResponse, error) {
	nodes, edges, err := ParseBlueprint(registry, blueprint)
	if err != nil {
		return nil, err
	}

	// Validate node configurations
	if err := ValidateNodes(nodes, registry); err != nil {
		return nil, err
	}

	orgID, _ := uuid.Parse(organizationID)
	now := time.Now()
	model := &models.Blueprint{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           blueprint.Name,
		Description:    blueprint.Description,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Nodes:          nodes,
		Edges:          edges,
	}

	if err := database.Conn().Create(model).Error; err != nil {
		return nil, err
	}

	return &pb.CreateBlueprintResponse{
		Blueprint: SerializeBlueprint(model),
	}, nil
}
