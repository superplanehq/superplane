package github

import (
	"fmt"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

func (g *GitHub) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "repository":
		client, err := common.NewClient(ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		repositories, err := client.ListRepositories()
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}

		return toIntegrationResources(repositories), nil
	default:
		return []core.IntegrationResource{}, nil
	}
}

func toIntegrationResources(repositories []*github.Repository) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(repositories))
	for _, repo := range repositories {
		resources = append(resources, core.IntegrationResource{
			Type: "repository",
			Name: repo.GetName(),
			ID:   fmt.Sprintf("%d", repo.GetID()),
		})
	}
	return resources
}
