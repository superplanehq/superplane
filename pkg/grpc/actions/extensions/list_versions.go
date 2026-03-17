package extensions

import (
	"context"

	"github.com/google/uuid"
	extensions "github.com/superplanehq/superplane/pkg/extensions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func ListVersions(ctx context.Context, storage *extensions.Storage, organizationID string, extensionID string) (*pb.ListVersionsResponse, error) {
	extension, err := models.FindExtension(uuid.MustParse(organizationID), extensionID)
	if err != nil {
		return nil, err
	}

	versions, err := extension.ListVersions()
	if err != nil {
		return nil, err
	}

	protoVersions := make([]*pb.ExtensionVersion, len(versions))
	for i, version := range versions {
		protoVersions[i] = SerializeVersion(&version)
	}

	return &pb.ListVersionsResponse{Versions: protoVersions}, nil
}
