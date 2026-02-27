package honeycomb

import "github.com/superplanehq/superplane/pkg/core"

func (h *Honeycomb) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	switch resourceType {
	case "dataset":
		datasets, err := client.ListDatasets()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(datasets))
		for _, d := range datasets {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: d.Name,
				ID:   d.Slug,
			})
		}
		return resources, nil

	case "trigger":
		datasetSlug := ctx.Parameters["datasetSlug"]
		if datasetSlug == "" {
			return []core.IntegrationResource{}, nil
		}
		triggers, err := client.ListTriggers(datasetSlug)
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(triggers))
		for _, t := range triggers {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: t.Name,
				ID:   t.ID,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}
