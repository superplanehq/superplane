package jira

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (j *Jira) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "project":
		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode metadata: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(metadata.Projects))
		for _, project := range metadata.Projects {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: fmt.Sprintf("%s (%s)", project.Name, project.Key),
				ID:   project.Key,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}
