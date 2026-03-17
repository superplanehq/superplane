package extensions

import (
	"context"

	"github.com/google/uuid"
	extensions "github.com/superplanehq/superplane/pkg/extensions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListExtensions(ctx context.Context, storage *extensions.Storage, organizationID uuid.UUID) (*pb.ListExtensionsResponse, error) {
	extensions, err := models.ListExtensions(organizationID)
	if err != nil {
		return nil, err
	}

	protoExtensions := make([]*pb.Extension, len(extensions))
	for i, extension := range extensions {
		protoExtensions[i] = SerializeExtension(&extension)
	}

	return &pb.ListExtensionsResponse{Extensions: protoExtensions}, nil
}

func SerializeExtension(extension *models.Extension) *pb.Extension {
	return &pb.Extension{
		Metadata: &pb.Extension_Metadata{
			Id:          extension.ID.String(),
			Name:        extension.Name,
			Description: extension.Description,
			CreatedAt:   timestamppb.New(*extension.CreatedAt),
		},
	}
}
