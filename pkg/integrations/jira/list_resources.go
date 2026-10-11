package jira

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (j *Jira) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "project":
		return listProjects(ctx)
	case "issueType":
		return listIssueTypes(ctx)
	case "issueStatus":
		return listIssueStatuses(ctx)
	case "assignee":
		return listAssignees(ctx)
	case "priority":
		return listPriorities(ctx)
	case "resolution":
		return listResolutions(ctx)
	case "issue":
		return listIssues(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listIssues(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	projectKey := strings.TrimSpace(ctx.Parameters["project"])

	var jql string
	if projectKey != "" {
		jql = fmt.Sprintf(`project = "%s" ORDER BY updated DESC`, jqlQuotedProjectKey(projectKey))
	} else {
		jql = "updated >= -90d ORDER BY updated DESC"
	}

	hits, err := client.SearchIssuesUpTo(jql, 500)
	if err != nil {
		return nil, fmt.Errorf("search issues: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(hits))
	for _, hit := range hits {
		if strings.TrimSpace(hit.Key) == "" {
			continue
		}
		name := hit.Key
		if hit.Fields != nil {
			if s, ok := hit.Fields["summary"].(string); ok && strings.TrimSpace(s) != "" {
				name = fmt.Sprintf("%s (%s)", strings.TrimSpace(s), hit.Key)
			}
		}
		resources = append(resources, core.IntegrationResource{
			Type: "issue",
			Name: name,
			ID:   hit.Key,
		})
	}
	return resources, nil
}

func listProjects(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if ctx.HTTP != nil {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err == nil {
			projects, err := client.ListProjects()
			if err == nil {
				return projectResources(projects), nil
			}
		}
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return projectResources(metadata.Projects), nil
}

func projectResources(projects []Project) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, project := range projects {
		resources = append(resources, core.IntegrationResource{
			Type: "project",
			Name: fmt.Sprintf("%s (%s)", project.Name, project.Key),
			ID:   project.Key,
		})
	}
	return resources
}

func listIssueTypes(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	projectKey := ctx.Parameters["project"]
	if projectKey == "" || strings.Contains(projectKey, "{{") {
		return []core.IntegrationResource{}, nil
	}

	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	issueTypes, err := client.GetProjectIssueTypes(projectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to list issue types: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(issueTypes))
	for _, it := range issueTypes {
		resources = append(resources, core.IntegrationResource{
			Type: "issueType",
			Name: it.Name,
			ID:   it.Name,
		})
	}
	return resources, nil
}

func listIssueStatuses(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	projectKey := strings.TrimSpace(ctx.Parameters["project"])
	if projectKey != "" && !strings.Contains(projectKey, "{{") {
		statuses, err := client.GetProjectStatuses(projectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to list issue statuses: %w", err)
		}
		return issueStatusResources(statuses), nil
	}

	statuses, err := client.ListGlobalStatuses()
	if err != nil {
		return nil, fmt.Errorf("failed to list issue statuses: %w", err)
	}
	return issueStatusResources(statuses), nil
}

func issueStatusResources(statuses []Status) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(statuses))
	for _, s := range statuses {
		resources = append(resources, core.IntegrationResource{
			Type: "issueStatus",
			Name: s.Name,
			ID:   s.Name,
		})
	}
	return resources
}

func listAssignees(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	projectKey := assigneeProjectKey(ctx)
	if projectKey == "" {
		return []core.IntegrationResource{}, nil
	}

	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	users, err := client.ListAssignableUsers(projectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignable users: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(users))
	for _, u := range users {
		name := u.DisplayName
		if u.EmailAddr != "" {
			name = fmt.Sprintf("%s (%s)", u.DisplayName, u.EmailAddr)
		}
		resources = append(resources, core.IntegrationResource{
			Type: "assignee",
			Name: name,
			ID:   u.AccountID,
		})
	}
	return resources, nil
}

func listPriorities(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	priorities, err := client.ListPriorities()
	if err != nil {
		return nil, fmt.Errorf("failed to list priorities: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(priorities))
	for _, p := range priorities {
		resources = append(resources, core.IntegrationResource{
			Type: "priority",
			Name: p.Name,
			ID:   p.Name,
		})
	}
	return resources, nil
}

func listResolutions(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	resolutions, err := client.ListResolutions()
	if err != nil {
		return nil, fmt.Errorf("failed to list resolutions: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(resolutions))
	for _, r := range resolutions {
		resources = append(resources, core.IntegrationResource{
			Type: "resolution",
			Name: r.Name,
			ID:   r.Name,
		})
	}
	return resources, nil
}

func assigneeProjectKey(ctx core.ListResourcesContext) string {
	projectKey := strings.TrimSpace(ctx.Parameters["project"])
	if projectKey != "" && !strings.Contains(projectKey, "{{") {
		return projectKey
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return ""
	}
	for _, p := range metadata.Projects {
		if k := strings.TrimSpace(p.Key); k != "" {
			return k
		}
	}
	return ""
}
