package extensions

import (
	"context"

	"github.com/google/uuid"
	extensions "github.com/superplanehq/superplane/pkg/extensions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func PublishVersion(ctx context.Context, storage *extensions.Storage, organizationID string, extensionID string, versionName string) (*pb.PublishVersionResponse, error) {
	extension, err := models.FindExtension(uuid.MustParse(organizationID), extensionID)
	if err != nil {
		return nil, err
	}

	version, err := extension.FindVersion(versionName)
	if err != nil {
		return nil, err
	}

	err = version.Publish()
	if err != nil {
		return nil, err
	}

	return &pb.PublishVersionResponse{Version: SerializeVersion(version)}, nil
}
