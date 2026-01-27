package pagerduty

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (p *PagerDuty) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "service":
		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode application metadata: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(metadata.Services))
		for _, service := range metadata.Services {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: service.Name,
				ID:   service.ID,
			})
		}
		return resources, nil

	case "priority":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		priorities, err := client.ListPriorities()
		if err != nil {
			return nil, fmt.Errorf("failed to list priorities: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(priorities))
		for _, priority := range priorities {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: priority.Name,
				ID:   priority.ID,
			})
		}
		return resources, nil

	case "user":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		users, err := client.ListUsers()
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(users))
		for _, user := range users {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: fmt.Sprintf("%s (%s)", user.Name, user.Email),
				ID:   user.ID,
			})
		}
		return resources, nil

	case "escalation_policy":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		policies, err := client.ListEscalationPolicies()
		if err != nil {
			return nil, fmt.Errorf("failed to list escalation policies: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(policies))
		for _, policy := range policies {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: policy.Name,
				ID:   policy.ID,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}
