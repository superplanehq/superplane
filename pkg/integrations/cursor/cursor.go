package cursor

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("cursor", &Cursor{})
}

type Cursor struct{}

type Configuration struct {
	LaunchAgentKey string `json:"launchAgentKey"`
	AdminKey       string `json:"adminKey"`
}

func (i *Cursor) Name() string {
	return "cursor"
}

func (i *Cursor) Label() string {
	return "Cursor"
}

func (i *Cursor) Icon() string {
	return "cpu"
}

func (i *Cursor) Description() string {
	return "Build workflows with Cursor AI Agents and track usage"
}

func (i *Cursor) Instructions() string {
	return "To get your API keys, visit the [Cursor Dashboard](https://cursor.com/dashboard). You may need separate keys for Agents and Admin features."
}

func (i *Cursor) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "launchAgentKey",
			Label:       "Cloud Agent API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Required for launching AI Agents.",
			Required:    false,
		},
		{
			Name:        "adminKey",
			Label:       "Admin API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "(For Teams) Required for fetching Usage Data.",
			Required:    false,
		},
	}
}

func (i *Cursor) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.LaunchAgentKey == "" && config.AdminKey == "" {
		return fmt.Errorf("one of the keys is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if config.LaunchAgentKey != "" {
		if err := client.VerifyLaunchAgent(); err != nil {
			return fmt.Errorf("cloud agent key verification failed: %w", err)
		}
	}

	if config.AdminKey != "" {
		if err := client.VerifyAdmin(); err != nil {
			return fmt.Errorf("admin key verification failed: %w", err)
		}
	}

	ctx.Integration.Ready()
	return nil
}

func (i *Cursor) Components() []core.Component {
	return []core.Component{
		&LaunchAgent{},
		&GetDailyUsageData{},
		&GetLastMessage{},
	}
}

func (i *Cursor) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (i *Cursor) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (i *Cursor) HandleRequest(ctx core.HTTPRequestContext) {}

func (i *Cursor) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType == "model" {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}

		models, err := client.ListModels()
		if err != nil {
			return nil, err
		}

		resources := []core.IntegrationResource{
			{Type: "model", ID: "", Name: "Auto (Recommended)"},
		}

		for _, model := range models {
			resources = append(resources, core.IntegrationResource{
				Type: "model",
				ID:   model,
				Name: model,
			})
		}

		return resources, nil
	}

	return []core.IntegrationResource{}, nil
}

func (i *Cursor) Actions() []core.Action {
	return []core.Action{}
}

func (i *Cursor) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
