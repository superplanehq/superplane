package semaphore

import "github.com/superplanehq/superplane/pkg/core"

func (s *Semaphore) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	if resourceType != "project" {
		return []core.ApplicationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return nil, err
	}

	projects, err := client.listProjects()
	if err != nil {
		return nil, err
	}

	resources := make([]core.ApplicationResource, 0, len(projects))
	for _, project := range projects {
		if project.Metadata == nil {
			continue
		}

		resources = append(resources, core.ApplicationResource{
			Type: resourceType,
			Name: project.Metadata.ProjectName,
			ID:   project.Metadata.ProjectID,
		})
	}

	return resources, nil
}
