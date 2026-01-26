package organizations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListIntegrationResources(ctx context.Context, registry *registry.Registry, orgID, integrationID, resourceType string) (*pb.ListIntegrationResourcesResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	ID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation ID")
	}

	i, err := models.FindAppInstallation(org, ID)
	if err != nil {
		return nil, err
	}

	integration, err := registry.GetIntegration(i.AppName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "integration %s not found", i.AppName)
	}

	appCtx := contexts.NewAppInstallationContext(
		database.Conn(),
		nil,
		i,
		registry.Encryptor,
		registry,
	)

	listCtx := core.ListResourcesContext{
		Logger: log.WithFields(log.Fields{
			"app_installation_id": i.ID.String(),
			"app_name":            i.AppName,
			"resource_type":       resourceType,
		}),
		HTTP:            contexts.NewHTTPContext(registry.GetHTTPClient()),
		AppInstallation: appCtx,
	}

	resources, err := integration.ListResources(resourceType, listCtx)
	if err != nil {
		log.WithError(err).Errorf("failed to list resources for integration %s", i.ID)
		return nil, status.Error(codes.Internal, "failed to list integration resources")
	}

	return &pb.ListIntegrationResourcesResponse{
		Resources: serializeIntegrationResources(resources),
	}, nil
}

func serializeIntegrationResources(resources []core.ApplicationResource) []*pb.IntegrationResourceRef {
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
