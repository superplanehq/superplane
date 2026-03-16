package extensions

import (
	"context"

	extensions "github.com/superplanehq/superplane/pkg/extensions"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func UpdateVersion(ctx context.Context, storage *extensions.Storage, organizationID string, extensionID string, versionID string, bundle []byte, digest string) (*pb.UpdateVersionResponse, error) {
	currentVersion, err := storage.FindVersionById(organizationID, extensionID, versionID)
	if err != nil {
		return nil, err
	}

	files, err := extensions.ExtractBundleFiles(bundle)
	if err != nil {
		return nil, err
	}

	newVersion := extensions.Version{
		Digest:       digest,
		State:        currentVersion.State,
		Version:      currentVersion.Version,
		ExtensionID:  currentVersion.ExtensionID,
		ID:           currentVersion.ID,
		Integrations: files.Manifest.Integrations,
		Components:   files.Manifest.Components,
		Triggers:     files.Manifest.Triggers,
	}

	err = storage.UpdateVersion(organizationID, extensionID, versionID, newVersion, files)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateVersionResponse{Version: SerializeVersion(&newVersion)}, nil
}
