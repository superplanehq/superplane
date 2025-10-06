package blueprints

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/blueprints"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateBlueprint(ctx context.Context, registry *registry.Registry, organizationID string, id string, blueprint *pb.Blueprint) (*pb.UpdateBlueprintResponse, error) {
	blueprintID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid blueprint id: %v", err)
	}

	nodes, edges, err := ParseBlueprint(registry, blueprint)
	if err != nil {
		return nil, err
	}

	// Validate node configurations
	if err := ValidateNodes(nodes, registry); err != nil {
		return nil, err
	}

	var existing models.Blueprint
	if err := database.Conn().Where("id = ? AND organization_id = ?", blueprintID, organizationID).First(&existing).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "blueprint not found: %v", err)
	}

	now := time.Now()
	existing.Name = blueprint.Name
	existing.Description = blueprint.Description
	existing.Nodes = nodes
	existing.Edges = edges
	existing.UpdatedAt = &now

	if err := database.Conn().Save(&existing).Error; err != nil {
		return nil, err
	}

	return &pb.UpdateBlueprintResponse{
		Blueprint: SerializeBlueprint(&existing),
	}, nil
}
