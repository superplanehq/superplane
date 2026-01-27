package slack

import (
	"github.com/superplanehq/superplane/pkg/core"
)

func (s *Slack) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "channel" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return nil, err
	}

	channels, err := client.ListChannels()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(channels))
	for _, channel := range channels {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: channel.Name,
			ID:   channel.ID,
		})
	}

	return resources, nil
}
