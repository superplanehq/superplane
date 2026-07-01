package organizations

import (
	"context"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
)

func DescribeIntegration(ctx context.Context, registry *registry.Registry, orgID, integrationID string) (*pb.DescribeIntegrationResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization ID")
	}

	integration, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid integration ID")
	}

	instance, err := models.FindIntegration(org, integration)
	if err != nil {
		return nil, err
	}

	nodeRefs, err := models.ListIntegrationNodeReferences(instance.ID)
	if err != nil {
		return nil, err
	}

	proto, err := serializeIntegration(registry, instance, nodeRefs)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeIntegrationResponse{
		Integration: proto,
	}, nil
}
