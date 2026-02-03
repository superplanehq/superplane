package organizations

import (
	"context"
	"net/url"
	"strings"

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

func ListIntegrationResources(
	ctx context.Context,
	registry *registry.Registry,
	orgID,
	integrationID,
	resourceType string,
	parameters string,
) (*pb.ListIntegrationResourcesResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	ID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation ID")
	}

	instance, err := models.FindIntegration(org, ID)
	if err != nil {
		return nil, err
	}

	integration, err := registry.GetIntegration(instance.AppName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "integration %s not found", instance.AppName)
	}

	integrationCtx := contexts.NewIntegrationContext(
		database.Conn(),
		nil,
		instance,
		registry.Encryptor,
		registry,
	)

	listCtx := core.ListResourcesContext{
		Logger: log.WithFields(log.Fields{
			"integration_id":   instance.ID.String(),
			"integration_name": instance.AppName,
			"resource_type":    resourceType,
		}),
		HTTP:        contexts.NewHTTPContext(registry.GetHTTPClient()),
		Integration: integrationCtx,
		Parameters:  parseParameters(parameters),
	}

	resources, err := integration.ListResources(resourceType, listCtx)
	if err != nil {
		log.WithError(err).Errorf("failed to list resources for integration %s", instance.ID)
		return nil, status.Error(codes.Internal, "failed to list integration resources")
	}

	return &pb.ListIntegrationResourcesResponse{
		Resources: serializeIntegrationResources(resources),
	}, nil
}

func parseParameters(parameters string) map[string]string {
	if parameters == "" {
		return nil
	}

	trimmed := strings.TrimPrefix(parameters, "?")
	values, err := url.ParseQuery(trimmed)
	if err != nil {
		log.WithError(err).Warn("invalid integration resource parameters")
		return nil
	}

	out := make(map[string]string, len(values))
	for name, entries := range values {
		if name == "" {
			continue
		}
		if len(entries) == 0 {
			continue
		}
		if len(entries) == 1 {
			out[name] = entries[0]
			continue
		}
		out[name] = strings.Join(entries, ",")
	}

	if len(out) == 0 {
		return nil
	}
	return out
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
