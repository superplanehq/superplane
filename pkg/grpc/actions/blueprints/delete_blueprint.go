package blueprints

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/blueprints"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteBlueprint(ctx context.Context, organizationID string, id string) (*pb.DeleteBlueprintResponse, error) {
	_, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid blueprint id: %v", err)
	}

	blueprint, err := models.FindBlueprint(organizationID, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "blueprint not found: %v", err)
	}

	if err := database.Conn().Delete(&blueprint).Error; err != nil {
		return nil, err
	}

	return &pb.DeleteBlueprintResponse{}, nil
}
