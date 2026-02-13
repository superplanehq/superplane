package gitlab

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeMember    = "member"
	ResourceTypeMilestone = "milestone"
	ResourceTypePipeline  = "pipeline"
	ResourceTypeProject   = "project"
)

func (g *GitLab) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeMember:
		return ListMembers(ctx)
	case ResourceTypeMilestone:
		return ListMilestones(ctx)
	case ResourceTypePipeline:
		return ListPipelines(ctx)
	case ResourceTypeProject:
		return ListProjects(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func ListProjects(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(metadata.Projects))
	for _, project := range metadata.Projects {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeProject,
			Name: project.Name,
			ID:   fmt.Sprintf("%d", project.ID),
		})
	}

	return resources, nil
}

func ListMembers(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	members, err := client.ListGroupMembers(client.groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(members))
	for _, m := range members {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeMember,
			Name: fmt.Sprintf("%s (@%s)", m.Name, m.Username),
			ID:   fmt.Sprintf("%d", m.ID),
		})
	}
	return resources, nil
}

func ListMilestones(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	projectID := ctx.Parameters["project"]
	if projectID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	milestones, err := client.ListMilestones(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list milestones: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(milestones))
	for _, m := range milestones {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeMilestone,
			Name: m.Title,
			ID:   fmt.Sprintf("%d", m.ID),
		})
	}
	return resources, nil
}

func ListPipelines(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	projectID := ctx.Parameters["project"]
	if projectID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	pipelines, err := client.ListPipelines(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(pipelines))
	for _, pipeline := range pipelines {
		label := fmt.Sprintf("#%d - %s - %s", pipeline.ID, pipeline.Status, pipeline.Ref)
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypePipeline,
			Name: label,
			ID:   fmt.Sprintf("%d", pipeline.ID),
		})
	}

	return resources, nil
}
