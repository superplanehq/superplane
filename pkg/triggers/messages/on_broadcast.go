package messages

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

type OnBroadcast struct{}

func init() {
	registry.RegisterTrigger("onBroadcast", &OnBroadcast{})
}

type OnBroadcastConfiguration struct {
	App string `json:"app" mapstructure:"app"`
}

func (c *OnBroadcast) Name() string {
	return "onBroadcast"
}

func (c *OnBroadcast) Label() string {
	return "On Broadcast"
}

func (c *OnBroadcast) Description() string {
	return "Receive broadcast messages from another SuperPlane app"
}

func (c *OnBroadcast) Color() string {
	return "gray"
}

func (c *OnBroadcast) Icon() string {
	return "rss"
}

func (c *OnBroadcast) Documentation() string {
	return "Receive broadcast messages from another SuperPlane app"
}

func (c *OnBroadcast) ExampleData() map[string]any {
	return map[string]any{
		"app": map[string]any{
			"id":   "123",
			"name": "Another App",
		},
		"node": map[string]any{
			"id":   "123",
			"name": "Node Name",
		},
		"payload": map[string]any{
			"message": "Hello, World!",
		},
	}
}

func (c *OnBroadcast) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "app",
			Label:       "App",
			Description: "The SuperPlane app to listen to",
			Type:        configuration.FieldTypeApp,
			Required:    true,
		},
	}
}

func (c *OnBroadcast) Setup(ctx core.TriggerContext) error {
	config := OnBroadcastConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.App == "" {
		return fmt.Errorf("app is required")
	}

	app, err := ctx.Apps.Get(config.App)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	return ctx.Apps.Subscribe(app.ID)
}

func (c *OnBroadcast) OnAppMessage(ctx core.AppMessageContext) error {
	return ctx.Events.Emit("app.broadcast", ctx.Message)
}

func (c *OnBroadcast) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *OnBroadcast) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (c *OnBroadcast) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 0, nil, nil
}

func (c *OnBroadcast) Cleanup(ctx core.TriggerContext) error {
	return nil
}
