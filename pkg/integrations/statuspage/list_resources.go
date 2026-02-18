package statuspage

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypePage                    = "page"
	ResourceTypeComponent               = "component"
	ResourceTypeIncident                = "incident"
	ResourceTypeImpact                  = "impact"
	ResourceTypeImpactUpdate            = "impact_update" // includes Don't override (__none__), maintenance for Update Incident
	ResourceTypeIncidentStatusRealtime  = "incident_status_realtime"
	ResourceTypeIncidentStatusScheduled = "incident_status_scheduled"
	ResourceTypeComponentStatus         = "component_status"
)

func (s *Statuspage) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypePage:
		return listPages(ctx)
	case ResourceTypeComponent:
		return listComponents(ctx)
	case ResourceTypeIncident:
		return listIncidents(ctx)
	case ResourceTypeImpact:
		return listImpactResources()
	case ResourceTypeImpactUpdate:
		return listImpactUpdateResources()
	case ResourceTypeIncidentStatusRealtime:
		return listIncidentStatusRealtimeResources()
	case ResourceTypeIncidentStatusScheduled:
		return listIncidentStatusScheduledResources()
	case ResourceTypeComponentStatus:
		return listComponentStatusResources()
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listImpactResources() ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{
		{Type: ResourceTypeImpact, Name: "None", ID: "none"},
		{Type: ResourceTypeImpact, Name: "Minor", ID: "minor"},
		{Type: ResourceTypeImpact, Name: "Major", ID: "major"},
		{Type: ResourceTypeImpact, Name: "Critical", ID: "critical"},
	}, nil
}

func listImpactUpdateResources() ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{
		{Type: ResourceTypeImpactUpdate, Name: "Don't override", ID: "__none__"},
		{Type: ResourceTypeImpactUpdate, Name: "None", ID: "none"},
		{Type: ResourceTypeImpactUpdate, Name: "Maintenance", ID: "maintenance"},
		{Type: ResourceTypeImpactUpdate, Name: "Minor", ID: "minor"},
		{Type: ResourceTypeImpactUpdate, Name: "Major", ID: "major"},
		{Type: ResourceTypeImpactUpdate, Name: "Critical", ID: "critical"},
	}, nil
}

func listIncidentStatusRealtimeResources() ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{
		{Type: ResourceTypeIncidentStatusRealtime, Name: "Investigating", ID: "investigating"},
		{Type: ResourceTypeIncidentStatusRealtime, Name: "Identified", ID: "identified"},
		{Type: ResourceTypeIncidentStatusRealtime, Name: "Monitoring", ID: "monitoring"},
		{Type: ResourceTypeIncidentStatusRealtime, Name: "Resolved", ID: "resolved"},
	}, nil
}

func listIncidentStatusScheduledResources() ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{
		{Type: ResourceTypeIncidentStatusScheduled, Name: "Scheduled", ID: "scheduled"},
		{Type: ResourceTypeIncidentStatusScheduled, Name: "In progress", ID: "in_progress"},
		{Type: ResourceTypeIncidentStatusScheduled, Name: "Verifying", ID: "verifying"},
		{Type: ResourceTypeIncidentStatusScheduled, Name: "Completed", ID: "completed"},
	}, nil
}

func listComponentStatusResources() ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{
		{Type: ResourceTypeComponentStatus, Name: "Operational", ID: "operational"},
		{Type: ResourceTypeComponentStatus, Name: "Degraded performance", ID: "degraded_performance"},
		{Type: ResourceTypeComponentStatus, Name: "Partial outage", ID: "partial_outage"},
		{Type: ResourceTypeComponentStatus, Name: "Major outage", ID: "major_outage"},
		{Type: ResourceTypeComponentStatus, Name: "Under maintenance", ID: "under_maintenance"},
	}, nil
}

func listPages(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	pages, err := client.ListPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list pages: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(pages))
	for _, page := range pages {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypePage,
			Name: page.Name,
			ID:   page.ID,
		})
	}
	return resources, nil
}

func listComponents(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	pageID := ctx.Parameters["page_id"]
	if pageID == "" || strings.Contains(pageID, "{{") {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	components, err := client.ListComponents(pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to list components: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(components))
	for _, comp := range components {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeComponent,
			Name: comp.Name,
			ID:   comp.ID,
		})
	}
	return resources, nil
}

// IncidentUseExpressionID is the sentinel value when user selects "Use expression" from the dropdown
// (shown when page is expression or invalid). The incidentExpression field then holds the actual expression.
const IncidentUseExpressionID = "__use_expression__"

func listIncidents(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	pageID := ctx.Parameters["page_id"]
	if pageID == "" || strings.Contains(pageID, "{{") {
		// Page is expression or not yet selected. Offer "Use expression" so user can type incident expression.
		return []core.IntegrationResource{
			{Type: ResourceTypeIncident, Name: "Use expression for incident", ID: IncidentUseExpressionID},
		}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	incidents, err := client.ListIncidents(pageID, "", 100)
	if err != nil {
		// When page_id is invalid (e.g. from expression or typo), API returns 404.
		// Return "Use expression" option so user can select it and type expression in incidentExpression field.
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return []core.IntegrationResource{
				{Type: ResourceTypeIncident, Name: "Use expression for incident", ID: IncidentUseExpressionID},
			}, nil
		}
		return nil, fmt.Errorf("failed to list incidents: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(incidents))
	for _, inc := range incidents {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeIncident,
			Name: inc.Name,
			ID:   inc.ID,
		})
	}
	return resources, nil
}
