package semaphore

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore/common"
)

func (s *Semaphore) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "project" {
		return []core.IntegrationResource{}, nil
	}

	client, err := common.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	projects, err := client.ListProjects()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, project := range projects {
		if project.Metadata == nil {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: project.Metadata.ProjectName,
			ID:   project.Metadata.ProjectID,
		})
	}

	return resources, nil
}
