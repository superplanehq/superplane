package extensions

import (
	"context"

	extensions "github.com/superplanehq/superplane/pkg/extensions"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func PublishVersion(ctx context.Context, storage *extensions.Storage, organizationID string, extensionID string, versionID string, version string) (*pb.PublishVersionResponse, error) {
	currentVersion, err := storage.FindVersionById(organizationID, extensionID, versionID)
	if err != nil {
		return nil, err
	}

	newVersion := extensions.Version{
		Version:      version,
		State:        "published",
		ExtensionID:  currentVersion.ExtensionID,
		ID:           currentVersion.ID,
		Digest:       currentVersion.Digest,
		Integrations: currentVersion.Integrations,
		Components:   currentVersion.Components,
		Triggers:     currentVersion.Triggers,
	}

	err = storage.UpdateVersion(organizationID, extensionID, versionID, newVersion, nil)
	if err != nil {
		return nil, err
	}

	return &pb.PublishVersionResponse{Version: SerializeVersion(&newVersion)}, nil
}
