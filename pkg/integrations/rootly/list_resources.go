package rootly

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (r *Rootly) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "service":
		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode application metadata: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(metadata.Services))
		for _, service := range metadata.Services {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: service.Name,
				ID:   service.ID,
			})
		}
		return resources, nil

	case "severity":
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
				Type: resourceType,
				Name: severity.Name,
				ID:   severity.ID,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}
