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

func DescribeIntegration(ctx context.Context, registry *registry.Registry, orgID, integrationID string) (*pb.DescribeIntegrationResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	integration, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid integration ID")
	}

	appInstallation, err := models.FindAppInstallation(org, integration)
	if err != nil {
		return nil, err
	}

	nodeRefs, err := models.ListAppInstallationNodeReferences(appInstallation.ID)
	if err != nil {
		return nil, err
	}

	proto, err := serializeIntegration(registry, appInstallation, nodeRefs)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeIntegrationResponse{
		Integration: proto,
	}, nil
}
