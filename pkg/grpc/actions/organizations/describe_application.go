package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescribeApplication(ctx context.Context, registry *registry.Registry, orgID, installationID string) (*pb.DescribeApplicationResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	installation, err := uuid.Parse(installationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation ID")
	}

	appInstallation, err := models.FindAppInstallation(org, installation)
	if err != nil {
		return nil, err
	}

	nodeRefs, err := models.ListAppInstallationNodeReferences(appInstallation.ID)
	if err != nil {
		return nil, err
	}

	proto, err := serializeAppInstallation(registry, appInstallation, nodeRefs)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeApplicationResponse{
		Installation: proto,
	}, nil
}
