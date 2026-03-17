package extensions

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	extensions "github.com/superplanehq/superplane/pkg/extensions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
	"gorm.io/gorm"
)

func UpdateVersion(ctx context.Context, storage *extensions.Storage, organizationID string, extensionID string, versionName string, bundle []byte, digest string) (*pb.UpdateVersionResponse, error) {
	files, err := extensions.ExtractBundleFiles(bundle)
	if err != nil {
		return nil, err
	}

	extension, err := models.FindExtension(uuid.MustParse(organizationID), extensionID)
	if err != nil {
		return nil, err
	}

	version, err := extension.FindVersion(versionName)
	if err != nil {
		return nil, err
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		err = version.UpdateInTransaction(tx, digest, files.Manifest)
		if err != nil {
			return err
		}

		err = storage.UploadVersion(organizationID, extension.Name, version.Name, files)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pb.UpdateVersionResponse{Version: SerializeVersion(version)}, nil
}
