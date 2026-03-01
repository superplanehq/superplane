package splitio

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("splitio", &SplitIO{})
}

type SplitIO struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

func (s *SplitIO) Name() string {
	return "splitio"
}

func (s *SplitIO) Label() string {
	return "Split"
}

func (s *SplitIO) Icon() string {
	return "splitio"
}

func (s *SplitIO) Description() string {
	return "Manage feature flags and react to flag changes in Split.io"
}

func (s *SplitIO) Instructions() string {
	return `## API integration

1. In the [Split Admin settings](https://app.split.io/admin/settings), navigate to **API Keys**.
2. Click **Add API Key** and select the **Admin** key type.
3. Give the key a name and create it.
4. Copy the API key and **paste it** in the Configuration section below.`
}

func (s *SplitIO) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "Admin API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Admin API key from Split.io. Create one in Admin settings > API Keys with the Admin key type.",
		},
	}
}

func (s *SplitIO) Components() []core.Component {
	return []core.Component{
		&GetFeatureFlag{},
	}
}

func (s *SplitIO) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnFeatureFlagChange{},
	}
}

func (s *SplitIO) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *SplitIO) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	if strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("Admin API key is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	_, err = client.ListWorkspaces()
	if err != nil {
		return fmt.Errorf("error validating Admin API key (listing workspaces): %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (s *SplitIO) HandleRequest(ctx core.HTTPRequestContext) {}

func (s *SplitIO) Actions() []core.Action {
	return []core.Action{}
}

func (s *SplitIO) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (s *SplitIO) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "workspace":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		workspaces, err := client.ListWorkspaces()
		if err != nil {
			return nil, fmt.Errorf("failed to list workspaces: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(workspaces))
		for _, w := range workspaces {
			resources = append(resources, core.IntegrationResource{
				Type: "workspace",
				Name: w.Name,
				ID:   w.ID,
			})
		}
		return resources, nil

	case "environment":
		workspaceID := ctx.Parameters["workspaceId"]
		if workspaceID == "" {
			return []core.IntegrationResource{}, nil
		}

		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		environments, err := client.ListEnvironments(workspaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to list environments: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(environments))
		for _, e := range environments {
			resources = append(resources, core.IntegrationResource{
				Type: "environment",
				Name: e.Name,
				ID:   e.ID,
			})
		}
		return resources, nil

	case "split":
		workspaceID := ctx.Parameters["workspaceId"]
		if workspaceID == "" {
			return []core.IntegrationResource{}, nil
		}

		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		splits, err := client.ListSplits(workspaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to list feature flags: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(splits))
		for _, sp := range splits {
			resources = append(resources, core.IntegrationResource{
				Type: "split",
				Name: sp.Name,
				ID:   sp.Name,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}
