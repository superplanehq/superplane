package extensions

import (
	"context"

	"github.com/google/uuid"
	extensions "github.com/superplanehq/superplane/pkg/extensions"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func CreateVersion(ctx context.Context, storage *extensions.Storage, organizationID string, extensionID string, bundle []byte, digest string) (*pb.CreateVersionResponse, error) {
	files, err := extensions.ExtractBundleFiles(bundle)
	if err != nil {
		return nil, err
	}

	version := extensions.Version{
		ID:           uuid.New().String(),
		ExtensionID:  extensionID,
		Digest:       digest,
		State:        "draft",
		Integrations: files.Manifest.Integrations,
		Components:   files.Manifest.Components,
		Triggers:     files.Manifest.Triggers,
	}

	err = storage.CreateVersion(organizationID, extensionID, version, files)
	if err != nil {
		return nil, err
	}

	return &pb.CreateVersionResponse{Version: SerializeVersion(&version)}, nil
}

func SerializeVersion(version *extensions.Version) *pb.ExtensionVersion {
	return &pb.ExtensionVersion{
		Metadata: &pb.ExtensionVersion_Metadata{
			Id:      version.ID,
			Version: version.Version,
		},
		Status: &pb.ExtensionVersion_Status{
			State: pb.ExtensionVersion_State(pb.ExtensionVersion_State_value[version.State]),
		},
	}
}
