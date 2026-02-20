package servicenow

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
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
		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode metadata: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(metadata.Categories))
		for _, choice := range metadata.Categories {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: choice.Label,
				ID:   choice.Value,
			})
		}

		return resources, nil

	case "assignment_group":
		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode metadata: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(metadata.AssignmentGroups))
		for _, group := range metadata.AssignmentGroups {
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

	case "state":
		return []core.IntegrationResource{
			{Type: resourceType, ID: "1", Name: "New"},
			{Type: resourceType, ID: "2", Name: "In Progress"},
			{Type: resourceType, ID: "3", Name: "On Hold"},
			{Type: resourceType, ID: "6", Name: "Resolved"},
			{Type: resourceType, ID: "7", Name: "Closed"},
			{Type: resourceType, ID: "8", Name: "Canceled"},
		}, nil

	case "urgency":
		return []core.IntegrationResource{
			{Type: resourceType, ID: "1", Name: "High"},
			{Type: resourceType, ID: "2", Name: "Medium"},
			{Type: resourceType, ID: "3", Name: "Low"},
		}, nil

	case "impact":
		return []core.IntegrationResource{
			{Type: resourceType, ID: "1", Name: "High"},
			{Type: resourceType, ID: "2", Name: "Medium"},
			{Type: resourceType, ID: "3", Name: "Low"},
		}, nil

	case "on_hold_reason":
		return []core.IntegrationResource{
			{Type: resourceType, ID: "1", Name: "Awaiting Caller"},
			{Type: resourceType, ID: "2", Name: "Awaiting Change"},
			{Type: resourceType, ID: "3", Name: "Awaiting Problem"},
			{Type: resourceType, ID: "4", Name: "Awaiting Vendor"},
		}, nil

	case "resolution_code":
		return []core.IntegrationResource{
			{Type: resourceType, ID: "Duplicate", Name: "Duplicate"},
			{Type: resourceType, ID: "Known error", Name: "Known error"},
			{Type: resourceType, ID: "No resolution provided", Name: "No resolution provided"},
			{Type: resourceType, ID: "Resolved by caller", Name: "Resolved by caller"},
			{Type: resourceType, ID: "Resolved by change", Name: "Resolved by change"},
			{Type: resourceType, ID: "Resolved by problem", Name: "Resolved by problem"},
			{Type: resourceType, ID: "Resolved by request", Name: "Resolved by request"},
			{Type: resourceType, ID: "Solution provided", Name: "Solution provided"},
			{Type: resourceType, ID: "Workaround provided", Name: "Workaround provided"},
			{Type: resourceType, ID: "User error", Name: "User error"},
		}, nil

	case "incident":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		incidents, err := client.ListIncidents(200)
		if err != nil {
			return nil, fmt.Errorf("failed to list incidents: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(incidents))
		for _, incident := range incidents {
			name := incident.Number
			if incident.ShortDescription != "" {
				name = fmt.Sprintf("%s - %s", incident.Number, incident.ShortDescription)
			}
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: name,
				ID:   incident.SysID,
			})
		}

		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}
