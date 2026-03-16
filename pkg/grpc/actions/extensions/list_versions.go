package extensions

import (
	"context"

	extensions "github.com/superplanehq/superplane/pkg/extensions"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func ListVersions(ctx context.Context, storage *extensions.Storage, organizationID string, extensionID string) (*pb.ListVersionsResponse, error) {
	versions, err := storage.ListVersions(organizationID, extensionID)
	if err != nil {
		return nil, err
	}

	protoVersions := make([]*pb.ExtensionVersion, len(versions))
	for i, version := range versions {
		protoVersions[i] = SerializeVersion(&version)
	}

	return &pb.ListVersionsResponse{Versions: protoVersions}, nil
}
