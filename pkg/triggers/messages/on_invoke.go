package messages

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

// TODO: call this just 'run'?
type OnInvoke struct{}

func init() {
	registry.RegisterTrigger("onInvoke", &OnInvoke{})
}

func (c *OnInvoke) Name() string {
	return "onInvoke"
}

func (c *OnInvoke) Label() string {
	return "On Invoke"
}

func (c *OnInvoke) Description() string {
	return "Handle invocations"
}

func (c *OnInvoke) Color() string {
	return "gray"
}

func (c *OnInvoke) Icon() string {
	return "play"
}

func (c *OnInvoke) Documentation() string {
	return ""
}

func (c *OnInvoke) ExampleData() map[string]any {
	return map[string]any{
		"app": map[string]any{
			"id":   "123",
			"name": "Caller App",
		},
		"node": map[string]any{
			"id":   "invoke",
			"name": "Invoke App",
		},
		"payload": map[string]any{
			"message": "Hello, World!",
		},
	}
}

func (c *OnInvoke) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *OnInvoke) Setup(ctx core.TriggerContext) error {
	return nil
}

func (c *OnInvoke) OnAppMessage(ctx core.AppMessageContext) error {
	return ctx.Events.Emit("app.invocation", ctx.Message)
}

func (c *OnInvoke) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *OnInvoke) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (c *OnInvoke) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 0, nil, nil
}

func (c *OnInvoke) Cleanup(ctx core.TriggerContext) error {
	return nil
}
