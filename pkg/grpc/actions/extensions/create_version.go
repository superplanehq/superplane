package extensions

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	extensions "github.com/superplanehq/superplane/pkg/extensions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func CreateVersion(ctx context.Context, storage *extensions.Storage, organizationID uuid.UUID, extensionID string, versionName string, bundle []byte, digest string) (*pb.CreateVersionResponse, error) {
	extension, err := models.FindExtension(organizationID, extensionID)
	if err != nil {
		return nil, err
	}

	files, err := extensions.ExtractBundleFiles(bundle)
	if err != nil {
		return nil, err
	}

	var version *models.ExtensionVersion
	database.Conn().Transaction(func(tx *gorm.DB) error {
		version, err = extension.CreateVersionInTransaction(tx, versionName, digest, files.Manifest)
		if err != nil {
			return err
		}

		err = storage.UploadVersion(version.OrganizationID.String(), extension.Name, version.Name, files)
		if err != nil {
			return err
		}

		return nil
	})

	return &pb.CreateVersionResponse{Version: SerializeVersion(version)}, nil
}

func SerializeVersion(version *models.ExtensionVersion) *pb.ExtensionVersion {
	return &pb.ExtensionVersion{
		Metadata: &pb.ExtensionVersion_Metadata{
			Id:        version.ID.String(),
			Version:   version.Name,
			CreatedAt: timestamppb.New(*version.CreatedAt),
		},
		Status: &pb.ExtensionVersion_Status{
			State: pb.ExtensionVersion_State(pb.ExtensionVersion_State_value[version.State]),
		},
	}
}
