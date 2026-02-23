package semaphore

import (
	"fmt"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeProject  = "project"
	ResourceTypePipeline = "pipeline"
)

func (s *Semaphore) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeProject:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}
		return listProjectResources(client, resourceType)
	case ResourceTypePipeline:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}
		return listPipelineResources(client, ctx, resourceType)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listProjectResources(client *Client, resourceType string) ([]core.IntegrationResource, error) {
	projects, err := client.listProjects()
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

func listPipelineResources(client *Client, ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	projectID := strings.TrimSpace(ctx.Parameters["project_id"])
	pipelineByID := map[string]core.IntegrationResource{}

	if projectID != "" {
		pipelines, err := client.ListPipelines(projectID)
		if err != nil {
			return nil, err
		}

		for _, pipeline := range pipelines {
			resource := pipelineToResource(pipeline, resourceType)
			if resource.ID == "" {
				continue
			}
			pipelineByID[resource.ID] = resource
		}

		return mapValues(pipelineByID), nil
	}

	projects, err := client.listProjects()
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		if project.Metadata == nil || strings.TrimSpace(project.Metadata.ProjectID) == "" {
			continue
		}

		pipelines, err := client.ListPipelines(project.Metadata.ProjectID)
		if err != nil {
			return nil, err
		}

		for _, pipeline := range pipelines {
			resource := pipelineToResource(pipeline, resourceType)
			if resource.ID == "" {
				continue
			}
			pipelineByID[resource.ID] = resource
		}
	}

	return mapValues(pipelineByID), nil
}

func pipelineToResource(pipeline PipelineSummary, resourceType string) core.IntegrationResource {
	id := strings.TrimSpace(pipeline.PipelineID)

	return core.IntegrationResource{
		Type: resourceType,
		Name: pipelineResourceName(pipeline, id),
		ID:   id,
	}
}

func pipelineResourceName(pipeline PipelineSummary, id string) string {
	name := strings.TrimSpace(pipeline.PipelineName)
	if name == "" {
		name = id
	}

	state := strings.TrimSpace(pipeline.State)

	if state != "" {
		return fmt.Sprintf("%s (%s)", name, state)
	}

	return name
}

func mapValues(resourcesByID map[string]core.IntegrationResource) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(resourcesByID))
	for _, resource := range resourcesByID {
		resources = append(resources, resource)
	}

	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Name == resources[j].Name {
			return resources[i].ID < resources[j].ID
		}
		return resources[i].Name < resources[j].Name
	})

	return resources
}
