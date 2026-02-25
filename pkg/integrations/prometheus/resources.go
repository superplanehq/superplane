package prometheus

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const ResourceTypeSilence = "silence"

func (p *Prometheus) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeSilence:
		return listSilenceResources(ctx, resourceType)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listSilenceResources(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	silences, err := client.ListSilences()
	if err != nil {
		return nil, fmt.Errorf("failed to list silences: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(silences))
	for _, silence := range silences {
		if strings.TrimSpace(silence.ID) == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: silenceResourceName(silence),
			ID:   silence.ID,
		})
	}

	return resources, nil
}

func silenceResourceName(silence AlertmanagerSilence) string {
	state := strings.TrimSpace(silence.Status.State)
	comment := strings.TrimSpace(silence.Comment)
	if comment != "" {
		if state == "" {
			return comment
		}
		return fmt.Sprintf("%s (%s)", comment, state)
	}

	if state == "" {
		return silence.ID
	}

	return fmt.Sprintf("%s (%s)", silence.ID, state)
}
