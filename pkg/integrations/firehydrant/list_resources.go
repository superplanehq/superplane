package firehydrant

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (f *FireHydrant) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "service":
		return listServicesResource(ctx)
	case "severity":
		return listSeveritiesResource(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listServicesResource(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(metadata.Services))
	for _, service := range metadata.Services {
		resources = append(resources, core.IntegrationResource{
			Type: "service",
			Name: service.Name,
			ID:   service.ID,
		})
	}
	return resources, nil
}

func listSeveritiesResource(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	severities, err := client.ListSeverities()
	if err != nil {
		return nil, fmt.Errorf("failed to list severities: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(severities))
	for _, severity := range severities {
		resources = append(resources, core.IntegrationResource{
			Type: "severity",
			Name: severity.Slug,
			ID:   severity.Slug,
		})
	}
	return resources, nil
}
