package cursor

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterApplication("cursor", &Cursor{})
}

type Cursor struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

func (c *Cursor) Name() string {
	return "cursor"
}

func (c *Cursor) Label() string {
	return "Cursor"
}

func (c *Cursor) Icon() string {
	return "bot"
}

func (c *Cursor) Description() string {
	return "Launch Cursor Cloud Agents to make code changes in repositories"
}

func (c *Cursor) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Cursor API key",
		},
	}
}

func (c *Cursor) Components() []core.Component {
	return []core.Component{
		&LaunchAgent{},
	}
}

func (c *Cursor) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (c *Cursor) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return err
	}

	ctx.AppInstallation.SetState("ready", "")
	return nil
}

func (c *Cursor) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (c *Cursor) CompareWebhookConfig(a, b any) (bool, error) {
	return true, nil
}

func (c *Cursor) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	return []core.ApplicationResource{}, nil
}

func (c *Cursor) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (c *Cursor) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
