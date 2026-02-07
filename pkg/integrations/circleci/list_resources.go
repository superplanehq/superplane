package circleci

import (
	"github.com/superplanehq/superplane/pkg/core"
)

func (c *CircleCI) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "project" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	projects, err := client.ListProjects()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, project := range projects {
		// CircleCI projects use slug as identifier (e.g., "gh/org/repo")
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: project.RepoName,
			ID:   project.Slug,
		})
	}

	return resources, nil
}
