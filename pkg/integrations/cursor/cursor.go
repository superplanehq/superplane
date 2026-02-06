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
	APIKey string `json:"apiKey" mapstructure:"apiKey"`
}

type WebhookConfiguration struct {
	Event string `json:"event" mapstructure:"event"`
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

func (c *Cursor) Instructions() string {
	return `Create a Cursor API key in the Cursor dashboard (Settings -> API Keys) and paste it below.`
}

func (c *Cursor) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Cursor API key (from Cursor dashboard -> Settings -> API Keys)",
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

func (c *Cursor) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *Cursor) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func (c *Cursor) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (c *Cursor) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.Event == configB.Event, nil
}

func (c *Cursor) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (c *Cursor) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	// Cursor does not register with an external webhook endpoint; our webhook URL is the callback.
	// Webhook URL and secret are provided to the Launch Agent component via IntegrationContext.GetWebhookURL/GetWebhookSecret at execution time.
	return nil, nil
}

func (c *Cursor) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}

func (c *Cursor) Actions() []core.Action {
	return []core.Action{}
}

func (c *Cursor) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
