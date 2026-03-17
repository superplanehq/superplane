package extensions

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/extensions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func CreateExtension(ctx context.Context, storage *extensions.Storage, organizationID uuid.UUID, name string, description string) (*pb.CreateExtensionResponse, error) {
	extension, err := models.CreateExtension(organizationID, name, description)
	if err != nil {
		return nil, err
	}

	return &pb.CreateExtensionResponse{Extension: SerializeExtension(extension)}, nil
}
