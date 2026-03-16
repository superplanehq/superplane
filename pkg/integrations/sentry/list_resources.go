package sentry

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

func (s *Sentry) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	switch resourceType {
	case "project":
		return listProjects(client)
	case "issue":
		projectSlug := ctx.Parameters["project"]
		if projectSlug == "" {
			return []core.IntegrationResource{}, nil
		}
		return listIssues(client, projectSlug)
	case "member":
		return listMembers(client)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listProjects(client *Client) ([]core.IntegrationResource, error) {
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

func listIssues(client *Client, projectSlug string) ([]core.IntegrationResource, error) {
	issues, err := client.ListIssues(projectSlug, &ListIssuesOptions{
		Query:   "is:unresolved",
		PerPage: 100,
	})
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, len(issues))
	for i, issue := range issues {
		displayName := fmt.Sprintf("%s: %s", issue.ShortID, issue.Title)
		resources[i] = core.IntegrationResource{
			Type: "issue",
			Name: displayName,
			ID:   issue.ID,
		}
	}

	return resources, nil
}

func listMembers(client *Client) ([]core.IntegrationResource, error) {
	members, err := client.ListOrganizationMembers()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(members))
	for _, member := range members {
		name := member.Name
		email := member.Email
		userID := member.ID

		if member.User != nil {
			if member.User.Name != "" {
				name = member.User.Name
			}
			if member.User.Email != "" {
				email = member.User.Email
			}
			userID = member.User.ID
		}

		displayName := name
		if email != "" && name != email {
			displayName = fmt.Sprintf("%s (%s)", name, email)
		}

		resources = append(resources, core.IntegrationResource{
			Type: "member",
			Name: displayName,
			ID:   userID,
		})
	}

	return resources, nil
}
