package teams

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

func (t *Teams) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "channel" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("Teams: failed to create client: %v", err)
		}
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	teams, err := client.ListTeams()
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("Teams: failed to list teams: %v", err)
		}
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	if ctx.Logger != nil {
		ctx.Logger.Infof("Teams: found %d teams", len(teams))
	}

	var resources []core.IntegrationResource
	for _, team := range teams {
		if ctx.Logger != nil {
			ctx.Logger.Infof("Teams: fetching channels for team %s (%s)", team.ID, team.DisplayName)
		}

		channels, err := client.ListTeamChannels(team.ID)
		if err != nil {
			if ctx.Logger != nil {
				ctx.Logger.Warnf("Teams: failed to get channels for team %s: %v", team.ID, err)
			}
			continue
		}

		if ctx.Logger != nil {
			ctx.Logger.Infof("Teams: found %d channels in team %s", len(channels), team.DisplayName)
		}

		for _, channel := range channels {
			resources = append(resources, core.IntegrationResource{
				Type: "channel",
				ID:   channel.ID,
				Name: fmt.Sprintf("#%s (%s)", channel.DisplayName, team.DisplayName),
			})
		}
	}

	if ctx.Logger != nil {
		ctx.Logger.Infof("Teams: returning %d channel resources", len(resources))
	}

	return resources, nil
}
