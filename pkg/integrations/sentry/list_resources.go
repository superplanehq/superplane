package sentry

import (
	"github.com/superplanehq/superplane/pkg/core"
)

func (s *Sentry) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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

	resources := make([]core.IntegrationResource, len(projects))
	for i, project := range projects {
		resources[i] = core.IntegrationResource{
			Type: "project",
			Name: project.Slug,
			ID:   project.ID,
		}
	}

	return resources, nil
}
