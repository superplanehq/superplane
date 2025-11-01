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
	"gorm.io/datatypes"
)

func UpdateBlueprint(ctx context.Context, registry *registry.Registry, organizationID string, id string, blueprint *pb.Blueprint) (*pb.UpdateBlueprintResponse, error) {
	_, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid blueprint id: %v", err)
	}

	existing, err := models.FindBlueprint(organizationID, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "blueprint not found: %v", err)
	}

	nodes, edges, err := ParseBlueprint(registry, blueprint)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid blueprint: %v", err)
	}

	outputChannels, err := ParseOutputChannels(registry, blueprint.Nodes, blueprint.OutputChannels)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid output channels: %v", err)
	}

	err = ValidateNodeConfigurations(nodes, registry)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid node configurations: %v", err)
	}

	configuration, err := ProtoToConfiguration(blueprint.Configuration)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid configuration: %v", err)
	}

	now := time.Now()
	existing.Name = blueprint.Name
	existing.Description = blueprint.Description
	existing.Icon = blueprint.Icon
	existing.Color = blueprint.Color
	existing.Nodes = nodes
	existing.Edges = edges
	existing.Configuration = datatypes.NewJSONSlice(configuration)
	existing.UpdatedAt = &now
	existing.OutputChannels = datatypes.NewJSONSlice(outputChannels)

	if err := database.Conn().Save(&existing).Error; err != nil {
		return nil, err
	}

	return &pb.UpdateBlueprintResponse{
		Blueprint: SerializeBlueprint(existing),
	}, nil
}
