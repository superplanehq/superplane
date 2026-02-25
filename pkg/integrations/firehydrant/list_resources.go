package firehydrant

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

func (f *FireHydrant) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "severity":
		return listResourcesForSeverity(ctx)
	case "priority":
		return listResourcesForPriority(ctx)
	case "service":
		return listResourcesForService(ctx)
	case "team":
		return listResourcesForTeam(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listResourcesForSeverity(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	severities, err := client.ListSeverities()
	if err != nil {
		return nil, fmt.Errorf("failed to list severities: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(severities))
	for _, severity := range severities {
		resources = append(resources, core.IntegrationResource{
			Type: "severity",
			Name: severity.Slug,
			ID:   severity.Slug,
		})
	}
	return resources, nil
}

func listResourcesForPriority(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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
			Type: "priority",
			Name: priority.Slug,
			ID:   priority.Slug,
		})
	}
	return resources, nil
}

func listResourcesForService(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	services, err := client.ListServices()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(services))
	for _, service := range services {
		resources = append(resources, core.IntegrationResource{
			Type: "service",
			Name: service.Name,
			ID:   service.ID,
		})
	}
	return resources, nil
}

func listResourcesForTeam(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	teams, err := client.ListTeams()
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(teams))
	for _, team := range teams {
		resources = append(resources, core.IntegrationResource{
			Type: "team",
			Name: team.Name,
			ID:   team.ID,
		})
	}
	return resources, nil
}
