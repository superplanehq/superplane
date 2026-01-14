package pagerduty

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (p *PagerDuty) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	if resourceType != "service" {
		return []core.ApplicationResource{}, nil
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode application metadata: %w", err)
	}

	resources := make([]core.ApplicationResource, 0, len(metadata.Services))
	for _, service := range metadata.Services {
		resources = append(resources, core.ApplicationResource{
			Type: resourceType,
			Name: service.Name,
			ID:   service.ID,
		})
	}

	return resources, nil
}
