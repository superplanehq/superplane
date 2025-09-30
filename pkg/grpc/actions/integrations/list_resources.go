package integrations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListResources(ctx context.Context, registry *registry.Registry, domainType, domainID, idOrName, resourceType string) (*pb.ListResourcesResponse, error) {
	err := actions.ValidateUUIDs(idOrName)
	var integration *models.Integration
	if err != nil {
		integration, err = models.FindIntegrationByName(domainType, uuid.MustParse(domainID), idOrName)
	} else {
		integration, err = models.FindDomainIntegration(domainType, uuid.MustParse(domainID), uuid.MustParse(idOrName))
	}

	if err != nil {
		return nil, status.Error(codes.NotFound, "integration not found")
	}

	resourceManager, err := registry.NewResourceManager(ctx, integration)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create resource manager")
	}

	resources, err := resourceManager.List(ctx, resourceType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list resources")
	}

	return &pb.ListResourcesResponse{
		Resources: serializeResources(registry, resources),
	}, nil
}

func serializeResources(registry *registry.Registry, in []integrations.Resource) []*pb.Resource {
	out := make([]*pb.Resource, len(in))
	for i, r := range in {
		out[i] = &pb.Resource{
			Id:   r.Id(),
			Name: r.Name(),
			Type: r.Type(),
			Url:  r.URL(),
		}
	}

	return out
}
