package organizations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
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
		//
		// A serialization failure (e.g. the integration's app is no longer
		// registered) is a server-side inconsistency, not a client error.
		// Wrap it so the gateway sanitizer classifies it as Internal instead
		// of leaking an unhandled HTTP 500.
		//
		log.Errorf("failed to serialize integration %s: %v", instance.AppName, err)
		return nil, grpcerrors.Internal(err, "failed to serialize integration")
	}

	return &pb.DescribeIntegrationResponse{
		Integration: proto,
	}, nil
}
