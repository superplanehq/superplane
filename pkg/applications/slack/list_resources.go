package slack

import (
	"github.com/superplanehq/superplane/pkg/core"
)

func (s *Slack) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	if resourceType != "channel" {
		return []core.ApplicationResource{}, nil
	}

	client, err := NewClient(ctx.AppInstallation)
	if err != nil {
		return nil, err
	}

	channels, err := client.ListChannels()
	if err != nil {
		return nil, err
	}

	resources := make([]core.ApplicationResource, 0, len(channels))
	for _, channel := range channels {
		resources = append(resources, core.ApplicationResource{
			Type: resourceType,
			Name: channel.Name,
			ID:   channel.ID,
		})
	}

	return resources, nil
}
