package servicenow

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

func (s *ServiceNow) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "user":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		var users []UserRecord

		groupID := ctx.Parameters["assignmentGroup"]
		if groupID != "" {
			users, err = client.ListGroupMembers(groupID)
		} else {
			users, err = client.ListUsers()
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(users))
		for _, user := range users {
			name := user.Name
			if user.Email != "" {
				name = fmt.Sprintf("%s (%s)", user.Name, user.Email)
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: name,
				ID:   user.SysID,
			})
		}

		return resources, nil

	case "category":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		categories, err := client.ListCategories()
		if err != nil {
			return nil, fmt.Errorf("failed to list categories: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(categories))
		for _, choice := range categories {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: choice.Label,
				ID:   choice.Value,
			})
		}

		return resources, nil

	case "assignment_group":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		groups, err := client.ListAssignmentGroups()
		if err != nil {
			return nil, fmt.Errorf("failed to list assignment groups: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(groups))
		for _, group := range groups {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: group.Name,
				ID:   group.SysID,
			})
		}

		return resources, nil

	case "subcategory":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		category := ctx.Parameters["category"]
		choices, err := client.ListSubcategories(category)
		if err != nil {
			return nil, fmt.Errorf("failed to list subcategories: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(choices))
		for _, choice := range choices {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: choice.Label,
				ID:   choice.Value,
			})
		}

		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}
