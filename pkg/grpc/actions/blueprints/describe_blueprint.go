package blueprints

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/blueprints"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescribeBlueprint(ctx context.Context, registry *registry.Registry, organizationID string, id string) (*pb.DescribeBlueprintResponse, error) {
	blueprintID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid blueprint id: %v", err)
	}

	var blueprint models.Blueprint
	if err := database.Conn().Where("id = ? AND organization_id = ?", blueprintID, organizationID).First(&blueprint).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "blueprint not found: %v", err)
	}

	return &pb.DescribeBlueprintResponse{
		Blueprint: SerializeBlueprint(&blueprint),
	}, nil
}
