package cloudsmith

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

func (c *Cloudsmith) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "repository":
		return listRepositories(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listRepositories(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	repositories, err := client.ListRepositories()
	if err != nil {
		return nil, fmt.Errorf("error listing repositories: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(repositories))
	for _, repository := range repositories {
		name := repository.Name
		if name == "" {
			name = repository.Slug
		}

		resources = append(resources, core.IntegrationResource{
			Type: "repository",
			Name: fmt.Sprintf("%s/%s", repository.Namespace, name),
			ID:   fmt.Sprintf("%s/%s", repository.Namespace, repository.Slug),
		})
	}

	return resources, nil
}
