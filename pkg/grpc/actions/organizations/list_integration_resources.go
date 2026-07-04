package organizations

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

func ListIntegrationResources(ctx context.Context, registry *registry.Registry, orgID string, integrationID string, parameters map[string]string) (*pb.ListIntegrationResourcesResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization ID")
	}

	ID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid installation ID")
	}

	resourceType := parameters["type"]
	if resourceType == "" {
		return nil, grpcerrors.InvalidArgument(nil, "resource type is required")
	}

	instance, err := models.FindIntegration(org, ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "integration not found")
		}

		return nil, grpcerrors.Internal(err, "failed to load integration")
	}

	if instance.State == models.IntegrationStateError {
		log.WithFields(log.Fields{
			"integration_id":   instance.ID,
			"integration_name": instance.AppName,
		}).Warn("integration is in error state")
		return nil, grpcerrors.FailedPrecondition(nil, "integration is in error state")
	}

	if instance.State != models.IntegrationStateReady {
		return &pb.ListIntegrationResourcesResponse{
			Resources: []*pb.IntegrationResourceRef{},
		}, nil
	}

	integration, err := registry.GetIntegration(instance.AppName)
	if err != nil {
		return nil, grpcerrors.FailedPrecondition(nil, fmt.Sprintf("integration %s is unavailable", instance.AppName))
	}

	integrationCtx := contexts.NewIntegrationContext(
		database.Conn(),
		nil,
		instance,
		registry.Encryptor,
		registry,
		nil,
	)

	listCtx := core.ListResourcesContext{
		Logger: log.WithFields(log.Fields{
			"integration_id":   instance.ID.String(),
			"integration_name": instance.AppName,
			"resource_type":    resourceType,
		}),
		HTTP:        registry.HTTPContext(),
		Integration: integrationCtx,
		Parameters:  parameters,
	}

	resources, err := integration.ListResources(resourceType, listCtx)
	if err != nil {
		log.WithError(err).WithField("integration_id", instance.ID).Warn("failed to list integration resources")
		return nil, grpcerrors.FailedPrecondition(nil, "failed to list integration resources")
	}

	return &pb.ListIntegrationResourcesResponse{
		Resources: serializeIntegrationResources(resources),
	}, nil
}

func serializeIntegrationResources(resources []core.IntegrationResource) []*pb.IntegrationResourceRef {
	out := make([]*pb.IntegrationResourceRef, 0, len(resources))
	for _, resource := range resources {
		out = append(out, &pb.IntegrationResourceRef{
			Type: resource.Type,
			Name: resource.Name,
			Id:   resource.ID,
		})
	}
	return out
}
