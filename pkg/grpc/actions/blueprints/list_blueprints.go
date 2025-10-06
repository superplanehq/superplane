package blueprints

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/blueprints"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListBlueprints(ctx context.Context, registry *registry.Registry, organizationID string) (*pb.ListBlueprintsResponse, error) {
	var blueprints []models.Blueprint

	if err := database.Conn().Where("organization_id = ?", organizationID).Find(&blueprints).Error; err != nil {
		return nil, err
	}

	protoBlueprints := make([]*pb.Blueprint, len(blueprints))
	for i, blueprint := range blueprints {
		protoBlueprints[i] = SerializeBlueprint(&blueprint)
	}

	return &pb.ListBlueprintsResponse{
		Blueprints: protoBlueprints,
	}, nil
}
