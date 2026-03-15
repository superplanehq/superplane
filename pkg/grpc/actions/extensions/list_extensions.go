package extensions

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func ListExtensions(ctx context.Context, storage *ExtensionStorage, organizationID string) (*pb.ListExtensionsResponse, error) {
	extensions, err := storage.ListExtensions(organizationID)
	if err != nil {
		return nil, err
	}

	protoExtensions := make([]*pb.Extension, len(extensions))
	for i, extension := range extensions {
		protoExtensions[i] = SerializeExtension(&extension)
	}

	return &pb.ListExtensionsResponse{Extensions: protoExtensions}, nil
}

func SerializeExtension(extension *Extension) *pb.Extension {
	return &pb.Extension{
		Metadata: &pb.Extension_Metadata{
			Id:          extension.ID,
			Name:        extension.Name,
			Description: extension.Description,
		},
	}
}
