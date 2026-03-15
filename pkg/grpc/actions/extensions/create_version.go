package extensions

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
)

func CreateVersion(ctx context.Context, storage *ExtensionStorage, organizationID string, extensionID string, bundle []byte, digest string) (*pb.CreateVersionResponse, error) {
	manifest, err := extractManifestFromBundle(bundle)
	if err != nil {
		return nil, err
	}

	version := Version{
		ID:           uuid.New().String(),
		ExtensionID:  extensionID,
		Digest:       digest,
		State:        "draft",
		Integrations: manifest.Integrations,
		Components:   manifest.Components,
		Triggers:     manifest.Triggers,
	}

	err = storage.CreateVersion(organizationID, extensionID, version)
	if err != nil {
		return nil, err
	}

	return &pb.CreateVersionResponse{Version: SerializeVersion(&version)}, nil
}

func SerializeVersion(version *Version) *pb.ExtensionVersion {
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
