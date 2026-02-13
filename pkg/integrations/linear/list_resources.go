package linear

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (l *Linear) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "team":
		metadata, err := decodeMetadata(ctx)
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(metadata.Teams))
		for _, team := range metadata.Teams {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: fmt.Sprintf("%s (%s)", team.Name, team.Key),
				ID:   team.ID,
			})
		}
		return resources, nil
	case "label":
		metadata, err := decodeMetadata(ctx)
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(metadata.Labels))
		for _, label := range metadata.Labels {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: label.Name,
				ID:   label.ID,
			})
		}
		return resources, nil
	case "state":
		teamID := ctx.Parameters["team"]
		if teamID == "" {
			return []core.IntegrationResource{}, nil
		}
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("create client: %w", err)
		}
		states, err := client.ListWorkflowStates(teamID)
		if err != nil {
			return nil, fmt.Errorf("list workflow states: %w", err)
		}
		resources := make([]core.IntegrationResource, 0, len(states))
		for _, s := range states {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: s.Name,
				ID:   s.ID,
			})
		}
		return resources, nil
	case "member":
		teamID := ctx.Parameters["team"]
		if teamID == "" {
			return []core.IntegrationResource{}, nil
		}
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("create client: %w", err)
		}
		members, err := client.ListTeamMembers(teamID)
		if err != nil {
			return nil, fmt.Errorf("list team members: %w", err)
		}
		resources := make([]core.IntegrationResource, 0, len(members))
		for _, m := range members {
			displayName := memberDisplayLabel(m)
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: displayName,
				ID:   m.ID,
			})
		}
		return resources, nil
	default:
		return []core.IntegrationResource{}, nil
	}
}

// memberDisplayLabel returns the display label for a team member, matching Linear's UX
// (e.g. "Andrew Gonzales" rather than "cool.dev12701" or email).
func memberDisplayLabel(m Member) string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	// Avoid showing raw email as primary label
	if m.Name != "" && !strings.Contains(m.Name, "@") {
		return m.Name
	}
	if m.Email != "" {
		return m.Email
	}
	return "Unnamed"
}

func decodeMetadata(ctx core.ListResourcesContext) (*Metadata, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	return &metadata, nil
}
