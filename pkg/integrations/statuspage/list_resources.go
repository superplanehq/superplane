package statuspage

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypePage      = "page"
	ResourceTypeComponent = "component"
	ResourceTypeIncident  = "incident"
)

func (s *Statuspage) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypePage:
		return listPages(ctx)
	case ResourceTypeComponent:
		return listComponents(ctx)
	case ResourceTypeIncident:
		return listIncidents(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
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
	if pageID == "" {
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

func listIncidents(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	pageID := ctx.Parameters["page_id"]
	if pageID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	incidents, err := client.ListIncidents(pageID, "", 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list incidents: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(incidents))
	for _, inc := range incidents {
		name := inc.Name
		if name == "" {
			name = inc.ID
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeIncident,
			Name: name,
			ID:   inc.ID,
		})
	}
	return resources, nil
}
