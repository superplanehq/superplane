package planelet

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnEvent struct{}

type OnEventConfiguration struct {
	EventType string `json:"eventType" mapstructure:"eventType"`
}

type OnEventMetadata struct {
	SubscriptionID *string `json:"subscriptionID,omitempty" mapstructure:"subscriptionID,omitempty"`
}

func (t *OnEvent) Name() string {
	return "planelet.onEvent"
}

func (t *OnEvent) Label() string {
	return "On Planelet Event"
}

func (t *OnEvent) Description() string {
	return "Listen for events from a connected Planelet server"
}

func (t *OnEvent) Documentation() string {
	return `Triggers a workflow when a Planelet server emits an event.

## Use Cases

- React to events from custom services
- Process webhooks from internal tools
- Listen for state changes in external systems

## Configuration

- **Event Type**: Optional filter — only trigger on events matching this type. Leave empty to receive all events.

## Event Data

The event payload depends on what the Planelet server sends. It is passed through as-is.`
}

func (t *OnEvent) Icon() string {
	return "puzzle"
}

func (t *OnEvent) Color() string {
	return "gray"
}

func (t *OnEvent) ExampleData() map[string]any {
	return map[string]any{
		"type": "planelet.event",
		"data": map[string]any{
			"eventType": "example.event",
			"payload": map[string]any{
				"message": "Something happened",
			},
		},
		"timestamp": "2026-05-30T12:00:00Z",
	}
}

func (t *OnEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "eventType",
			Label:       "Event Type",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional: filter to only trigger on events of this type. Leave empty for all events.",
		},
	}
}

func (t *OnEvent) Setup(ctx core.TriggerContext) error {
	var metadata OnEventMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.SubscriptionID != nil {
		return nil
	}

	var config OnEventConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	subscriptionConfig := map[string]any{
		"type": "planelet_event",
	}
	if config.EventType != "" {
		subscriptionConfig["eventType"] = config.EventType
	}

	subscriptionID, err := ctx.Integration.Subscribe(subscriptionConfig)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	s := subscriptionID.String()
	return ctx.Metadata.Set(OnEventMetadata{
		SubscriptionID: &s,
	})
}

func (t *OnEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *OnEvent) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnEvent) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}
