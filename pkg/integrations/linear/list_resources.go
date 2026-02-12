package linear

import (
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (l *Linear) ListResources(ctx core.ListResourcesContext) ([]core.Resource, error) {
	if ctx.Type != "team" {
		return nil, nil
	}

	var metadata Metadata
	if err := mapstructure.Decode(ctx.Integration.Metadata(), &metadata); err != nil {
		return nil, err
	}

	resources := make([]core.Resource, len(metadata.Teams))
	for i, team := range metadata.Teams {
		resources[i] = core.Resource{
			ID:    team.ID,
			Name:  team.Name,
			Label: team.Name + " (" + team.Key + ")",
		}
	}

	return resources, nil
}
