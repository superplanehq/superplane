package gitlab

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeMember      = "member"
	ResourceTypeMilestone   = "milestone"
	ResourceTypeProject     = "project"
	ResourceTypeEnvironment = "environment"
)

func (g *GitLab) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeMember:
		return ListMembers(ctx)
	case ResourceTypeMilestone:
		return ListMilestones(ctx)
	case ResourceTypeProject:
		return ListProjects(ctx)
	case ResourceTypeEnvironment:
		return ListEnvironments(ctx)
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

	var members []User
	if client.groupID != "" {
		members, err = client.ListGroupMembers(client.groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to list members: %v", err)
		}
	} else if projectID := ctx.Parameters["project"]; projectID != "" {
		members, err = client.ListProjectMembers(projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to list members: %v", err)
		}
	} else {
		user, err := client.getCurrentUser()
		if err != nil {
			return nil, fmt.Errorf("failed to get current user: %v", err)
		}
		members = []User{*user}
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

func ListEnvironments(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	projectID := ctx.Parameters["project"]
	if projectID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	environments, err := client.ListEnvironments(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(environments))
	for _, e := range environments {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeEnvironment,
			Name: e.Name,
			ID:   fmt.Sprintf("%d", e.ID),
		})
	}
	return resources, nil
}
