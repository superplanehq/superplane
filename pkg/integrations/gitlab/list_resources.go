package gitlab

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (g *GitLab) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "repository" {
		return []core.IntegrationResource{}, nil
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(metadata.Repositories))
	for _, repo := range metadata.Repositories {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: repo.Name,
			ID:   fmt.Sprintf("%d", repo.ID),
		})
	}

	return resources, nil
}
