package dockerhub

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

func listDockerHubResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "dockerhub.repository" {
		return []core.IntegrationResource{}, nil
	}

	namespace, err := resolveNamespace(ctx.Parameters["namespace"], ctx.Integration)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	repositories, err := client.ListRepositories(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list Docker Hub repositories: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(repositories))
	for _, repository := range repositories {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: repository.Name,
			ID:   repository.Name,
		})
	}

	return resources, nil
}
