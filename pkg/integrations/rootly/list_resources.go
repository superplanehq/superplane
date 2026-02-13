package rootly

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (r *Rootly) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "incident_status":
		return listResourcesForIncidentStatus(ctx)
	case "service":
		return listResourcesForService(ctx)
	case "severity":
		return listResourcesForSeverity(ctx)
	case "team":
		return listResourcesForTeam(ctx)
	case "sub_status":
		return listResourcesForSubStatus(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listResourcesForService(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode application metadata: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(metadata.Services))
	for _, service := range metadata.Services {
		resources = append(resources, core.IntegrationResource{
			Type: "service",
			Name: service.Name,
			ID:   service.ID,
		})
	}
	return resources, nil
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
			Name: severity.Name,
			ID:   severity.ID,
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

func listResourcesForSubStatus(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	subStatuses, err := client.ListSubStatuses()
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-statuses: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(subStatuses))
	for _, subStatus := range subStatuses {
		resources = append(resources, core.IntegrationResource{
			Type: "sub_status",
			Name: subStatus.Name,
			ID:   subStatus.ID,
		})
	}
	return resources, nil
}

func listResourcesForIncidentStatus(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	statuses, err := client.ListStatuses()
	if err != nil {
		return nil, fmt.Errorf("failed to list statuses: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(statuses))
	for _, status := range statuses {
		if !status.Enabled {
			continue
		}

		resourceID := status.Slug
		if resourceID == "" {
			resourceID = status.ID
		}

		resourceName := status.Name
		if resourceName == "" {
			resourceName = resourceID
		}

		resources = append(resources, core.IntegrationResource{
			Type: "incident_status",
			Name: resourceName,
			ID:   resourceID,
		})
	}
	return resources, nil
}
