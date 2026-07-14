package messages

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

type BroadcastMessage struct{}

func init() {
	registry.RegisterAction("broadcastMessage", &BroadcastMessage{})
}

type BroadcastMessageConfiguration struct {
	Payload any `json:"payload" mapstructure:"payload"`
}

func (c *BroadcastMessage) Name() string {
	return "broadcastMessage"
}

func (c *BroadcastMessage) Label() string {
	return "Broadcast Message"
}

func (c *BroadcastMessage) Color() string {
	return "gray"
}

func (c *BroadcastMessage) Icon() string {
	return "rss"
}

func (c *BroadcastMessage) Documentation() string {
	return "Broadcast a message to other SuperPlane apps subscribed to this app"
}

func (c *BroadcastMessage) Description() string {
	return "Broadcast a message to other SuperPlane apps"
}

func (c *BroadcastMessage) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"message": "Hello, world!",
		},
	}
}

func (c *BroadcastMessage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *BroadcastMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *BroadcastMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "payload",
			Description: "The payload to broadcast",
			Type:        configuration.FieldTypeObject,
			Required:    true,
		},
	}
}

func (c *BroadcastMessage) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *BroadcastMessage) Execute(ctx core.ExecutionContext) error {
	config := BroadcastMessageConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("broadcast message: decode configuration: %w", err)
	}

	err = ctx.Apps.Broadcast(config.Payload)
	if err != nil {
		return fmt.Errorf("broadcast message: dispatch: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "app.broadcast", []any{config.Payload})
}

func (c *BroadcastMessage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *BroadcastMessage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *BroadcastMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 0, nil, nil
}

func (c *BroadcastMessage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *BroadcastMessage) Cleanup(ctx core.SetupContext) error {
	return nil
}
