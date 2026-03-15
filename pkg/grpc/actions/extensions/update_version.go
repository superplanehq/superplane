package extensions

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func UpdateVersion(ctx context.Context, storage *ExtensionStorage, organizationID string, extensionID string, versionID string, bundle []byte, digest string) (*pb.UpdateVersionResponse, error) {
	currentVersion, err := storage.FindVersionById(organizationID, extensionID, versionID)
	if err != nil {
		return nil, err
	}

	manifest, err := extractManifestFromBundle(bundle)
	if err != nil {
		return nil, err
	}

	newVersion := Version{
		Digest:       digest,
		State:        currentVersion.State,
		Version:      currentVersion.Version,
		ExtensionID:  currentVersion.ExtensionID,
		ID:           currentVersion.ID,
		Integrations: manifest.Integrations,
		Components:   manifest.Components,
		Triggers:     manifest.Triggers,
	}

	err = storage.UpdateVersion(organizationID, extensionID, versionID, newVersion)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateVersionResponse{Version: SerializeVersion(&newVersion)}, nil
}
