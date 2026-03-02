package honeycomb

import "github.com/superplanehq/superplane/pkg/core"

const allDatasetsInEnvironmentScopeSlug = "__all__"

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
		triggers, err := listDatasetAndEnvironmentTriggers(client, datasetSlug)
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

func listDatasetAndEnvironmentTriggers(client *Client, datasetSlug string) ([]HoneycombTrigger, error) {
	triggers, err := client.ListTriggers(datasetSlug)
	if err != nil {
		return nil, err
	}

	if datasetSlug == allDatasetsInEnvironmentScopeSlug {
		return triggers, nil
	}

	environmentTriggers, err := client.ListTriggers(allDatasetsInEnvironmentScopeSlug)
	if err != nil {
		return triggers, nil
	}

	seen := map[string]struct{}{}
	merged := make([]HoneycombTrigger, 0, len(triggers)+len(environmentTriggers))

	for _, trigger := range triggers {
		key := trigger.ID
		if key == "" {
			key = trigger.Name
		}
		seen[key] = struct{}{}
		merged = append(merged, trigger)
	}

	for _, trigger := range environmentTriggers {
		key := trigger.ID
		if key == "" {
			key = trigger.Name
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, trigger)
	}

	return merged, nil
}
