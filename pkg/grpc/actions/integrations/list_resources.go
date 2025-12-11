package integrations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListResources(ctx context.Context, registry *registry.Registry, domainType, domainID, idOrName string, resourceType string) (*pb.ListResourcesResponse, error) {
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

	log.Infof("AAAAAAAAAAAAAAA: Listing resources for integration %s of type %s", integration.Name, integration.Type)

	resourceManager, err := registry.NewResourceManager(ctx, integration)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create resource manager")
	}

	log.Infof("AAAAAAAAAAAAAAA: Resource manager created for integration %s of type %s", integration.Name, integration.Type)

	resources, err := resourceManager.List(resourceType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list resources")
	}

	response := &pb.ListResourcesResponse{
		Resources: serializeResources(resources),
	}

	return response, nil
}

func serializeResources(resources []integrations.Resource) []*pb.ResourceRef {
	out := []*pb.ResourceRef{}

	for _, r := range resources {
		out = append(out, &pb.ResourceRef{
			Type: r.Type(),
			Name: r.Name(),
			Id:   r.Id(),
		})
	}

	return out
}
