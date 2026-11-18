package messages

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"

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
	App     string `json:"app" mapstructure:"app"`
	Enabled *bool  `json:"enabled" mapstructure:"enabled"`
	Filter  string `json:"filter" mapstructure:"filter"`
}

type OnBroadcastMetadata struct {
	App *AppMetadata `json:"app,omitempty" mapstructure:"app,omitempty"`
}

type AppMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
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
	return `Receive messages from the configured source app.

The optional filter is a bare boolean expression. Use message.app, message.node, and message.payload.
For example: message.payload.type == "completed".

A blank filter accepts all messages. True emits the original message unchanged.
False acknowledges delivery without a trigger event or run. Invalid expressions fail setup.
Runtime evaluation errors fail delivery and use the existing transaction retry path.
Run functions and node outputs are unavailable before a run exists.

Disable Enabled to pause new deliveries.`
}

func (c *OnBroadcast) ExampleData() map[string]any {
	return map[string]any{
		"type":      "app.broadcast",
		"timestamp": "2024-01-01T09:00:00Z",
		"data": map[string]any{
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
		{Name: "enabled", Label: "Enabled", Type: configuration.FieldTypeBool, Default: true},
		{Name: "filter", Label: "Filter", Description: "Accept messages when this expression is true. Use message.app, message.node, and message.payload.", Type: configuration.FieldTypeExpression},
	}
}

func (c *OnBroadcast) Setup(ctx core.TriggerContext) error {
	config := OnBroadcastConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if _, err := compileBroadcastFilter(config.Filter); err != nil {
		return err
	}

	metadata := OnBroadcastMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.App != nil {
		return c.Update(ctx, config, metadata)
	}

	return c.Create(ctx, config)
}

func (c *OnBroadcast) Create(ctx core.TriggerContext, config OnBroadcastConfiguration) error {
	if config.App == "" {
		return fmt.Errorf("app is required")
	}

	app, err := ctx.Apps.Get(config.App)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	if err := ctx.Apps.Subscribe(config.App); err != nil {
		return fmt.Errorf("failed to subscribe to app: %w", err)
	}

	return ctx.Metadata.Set(OnBroadcastMetadata{
		App: &AppMetadata{
			ID:   app.ID,
			Name: app.Name,
		},
	})
}

func (c *OnBroadcast) Update(ctx core.TriggerContext, config OnBroadcastConfiguration, metadata OnBroadcastMetadata) error {
	if metadata.App.ID == config.App || metadata.App.Name == config.App {
		return nil
	}

	// Clear the metadata
	err := ctx.Metadata.Set(OnBroadcastMetadata{})
	if err != nil {
		return fmt.Errorf("failed to clear metadata: %w", err)
	}

	// Unsubscribe from the current app
	err = ctx.Apps.Unsubscribe()
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from current app: %w", err)
	}

	if config.App == "" {
		return fmt.Errorf("app is required")
	}

	app, err := ctx.Apps.Get(config.App)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	if err := ctx.Apps.Subscribe(config.App); err != nil {
		return fmt.Errorf("failed to subscribe to app: %w", err)
	}

	return ctx.Metadata.Set(OnBroadcastMetadata{
		App: &AppMetadata{
			ID:   app.ID,
			Name: app.Name,
		},
	})
}

func (c *OnBroadcast) OnAppMessage(ctx core.AppMessageContext) error {
	var config OnBroadcastConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("decode broadcast configuration: %w", err)
	}
	if config.Enabled != nil && !*config.Enabled {
		return nil
	}
	program, err := compileBroadcastFilter(config.Filter)
	if err != nil {
		return err
	}
	if program != nil {
		result, err := expr.Run(program, map[string]any{"message": ctx.Message})
		if err != nil {
			return fmt.Errorf("evaluate broadcast filter: %w", err)
		}
		matches, ok := result.(bool)
		if !ok {
			return fmt.Errorf("broadcast filter must return a boolean, got %T", result)
		}
		if !matches {
			return nil
		}
	}
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
	return ctx.Apps.Unsubscribe()
}

// Only message is available before a run exists. Do not register run functions.
func compileBroadcastFilter(filter string) (*vm.Program, error) {
	if strings.TrimSpace(filter) == "" {
		return nil, nil
	}
	program, err := expr.Compile(filter, expr.Env(map[string]any{"message": map[string]any{"app": map[string]any{}, "node": map[string]any{}, "payload": map[string]any{}}}), expr.AsBool())
	if err != nil {
		return nil, fmt.Errorf("invalid broadcast filter: %w", err)
	}
	return program, nil
}
