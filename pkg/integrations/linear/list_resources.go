package linear

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (l *Linear) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "team":
		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode metadata: %w", err)
		}
		resources := make([]core.IntegrationResource, 0, len(metadata.Teams))
		for _, team := range metadata.Teams {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: fmt.Sprintf("%s (%s)", team.Name, team.Key),
				ID:   team.ID,
			})
		}
		return resources, nil
	case "label":
		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode metadata: %w", err)
		}
		resources := make([]core.IntegrationResource, 0, len(metadata.Labels))
		for _, label := range metadata.Labels {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: label.Name,
				ID:   label.ID,
			})
		}
		return resources, nil
	default:
		return []core.IntegrationResource{}, nil
	}
}
