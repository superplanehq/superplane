package extensions

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func CreateExtension(ctx context.Context, storage *ExtensionStorage, organizationID string, name string, description string) (*pb.CreateExtensionResponse, error) {
	extension := Extension{
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
