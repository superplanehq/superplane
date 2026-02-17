package dockerhub

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeRepository = "dockerhub.repository"
)

func listDockerHubResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeRepository:
		return listDockerHubRepositories(ctx)

	default:
		return []core.IntegrationResource{}, nil
	}
}

func listDockerHubRepositories(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	namespace, err := ctx.Integration.GetConfig("username")
	if err != nil {
		return nil, fmt.Errorf("integration username is required: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	repositories, err := client.ListRepositories(string(namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(repositories))
	for _, repository := range repositories {
		name := repository.Namespace + "/" + repository.Name
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeRepository,
			Name: name,
			ID:   name,
		})
	}

	return resources, nil
}
