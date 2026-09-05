package factory

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const OnWorkOrderTriggerName = "onWorkOrder"
const OnWorkOrderPayloadType = "workOrder.created"

func init() {
	registry.RegisterTrigger(OnWorkOrderTriggerName, &OnWorkOrder{})
}

// OnWorkOrder starts a workflow when a task is created in the factory
// that owns this app. SuperPlane emits the event; the trigger has no webhook.
type OnWorkOrder struct{}

func (t *OnWorkOrder) Name() string {
	return OnWorkOrderTriggerName
}

func (t *OnWorkOrder) Label() string {
	return "On Task"
}

func (t *OnWorkOrder) Description() string {
	return "Start when a task is created in this factory"
}

func (t *OnWorkOrder) Documentation() string {
	return `The On Task trigger starts a workflow when a task is created in the factory that owns this app.

Use it on a factory-owned app, such as the generated Backlog automation that scores new tasks.

## Event Data

Each event has type ` + "`workOrder.created`" + `. The payload is:

` + "```" + `
{
  "workOrder": {
    "id": "...",
    "title": "...",
    "description": "...",
    "number": 12,
    "state": "draft",
    "origin": { "url": "...", "label": "..." }
  }
}
` + "```" + `

` + "`origin`" + ` is present only when the task was created from an intake item.`
}

func (t *OnWorkOrder) Icon() string {
	return "factory"
}

func (t *OnWorkOrder) Color() string {
	return "blue"
}

func (t *OnWorkOrder) ExampleData() map[string]any {
	return map[string]any{
		"type":      OnWorkOrderPayloadType,
		"timestamp": "2026-01-01T00:00:00Z",
		"data": map[string]any{
			"workOrder": map[string]any{
				"id":          "123",
				"title":       "Show a clearer empty state",
				"description": "The billing page empty state does not name the next action.",
				"number":      12,
				"state":       "draft",
				"origin": map[string]any{
					"url":   "https://github.com/acme/app/issues/42",
					"label": "acme/app#42",
				},
			},
		},
	}
}

func (t *OnWorkOrder) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnWorkOrder) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *OnWorkOrder) Setup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnWorkOrder) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnWorkOrder) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, fmt.Errorf("hook %s not supported", ctx.Name)
}

func (t *OnWorkOrder) Cleanup(ctx core.TriggerContext) error {
	return nil
}
