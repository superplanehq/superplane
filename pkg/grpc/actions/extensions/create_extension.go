package extensions

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/extensions"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func CreateExtension(ctx context.Context, storage *extensions.Storage, organizationID string, name string, description string) (*pb.CreateExtensionResponse, error) {
	extension := extensions.Extension{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
	}

	err := storage.CreateExtension(organizationID, extension)
	if err != nil {
		return nil, err
	}

	return &pb.CreateExtensionResponse{Extension: SerializeExtension(&extension)}, nil
}
