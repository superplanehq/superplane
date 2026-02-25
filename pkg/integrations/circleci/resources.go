package circleci

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeProject  = "project"
	ResourceTypeWorkflow = "workflow"
)

func ListProjectSlugs(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	collaborations, err := client.GetCollaborations()
	if err != nil {
		return nil, fmt.Errorf("failed to list collaborations: %v", err)
	}

	seen := make(map[string]bool)
	resources := []core.IntegrationResource{}

	for _, collab := range collaborations {
		if collab.Slug == "" {
			continue
		}

		pipelines, err := client.GetPipelinesByOrg(collab.Slug)
		if err != nil {
			continue
		}

		for _, p := range pipelines.Items {
			if p.ProjectSlug == "" || seen[p.ProjectSlug] {
				continue
			}
			seen[p.ProjectSlug] = true

			name := p.ProjectSlug
			if strings.HasPrefix(p.ProjectSlug, "circleci/") {
				project, err := client.GetProject(p.ProjectSlug)
				if err == nil && project.Name != "" {
					name = fmt.Sprintf("%s/%s", project.OrganizationName, project.Name)
				}
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeProject,
				Name: name,
				ID:   p.ProjectSlug,
			})
		}
	}

	return resources, nil
}

func ListWorkflowNames(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	projectSlug := ctx.Parameters["projectSlug"]
	if projectSlug == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	summaries, err := client.ListWorkflowSummaries(projectSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %v", err)
	}

	seen := make(map[string]bool)
	resources := []core.IntegrationResource{}
	for _, w := range summaries.Items {
		if w.Name == "" || seen[w.Name] {
			continue
		}
		seen[w.Name] = true
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeWorkflow,
			Name: w.Name,
			ID:   w.Name,
		})
	}

	return resources, nil
}
