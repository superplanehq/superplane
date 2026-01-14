package github

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (g *GitHub) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	if resourceType != "repository" {
		return []core.ApplicationResource{}, nil
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode application metadata: %w", err)
	}

	resources := make([]core.ApplicationResource, 0, len(metadata.Repositories))
	for _, repo := range metadata.Repositories {
		resources = append(resources, core.ApplicationResource{
			Type: resourceType,
			Name: repo.Name,
			ID:   fmt.Sprintf("%d", repo.ID),
		})
	}

	return resources, nil
}
