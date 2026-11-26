package factory

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const OnWebhookTriggerName = "onWebhook"
const OnWebhookPayloadType = "webhook"

func init() {
	registry.RegisterTrigger(OnWebhookTriggerName, &OnWebhook{})
}

// OnWebhook starts a workflow when a workspace webhook receives a request.
// The public receiver authenticates the request once for every listener, so
// this trigger only parses and emits the payload.
type OnWebhook struct{}

type OnWebhookConfiguration struct {
	Webhook string `json:"webhook" mapstructure:"webhook"`
}

func (t *OnWebhook) Name() string {
	return OnWebhookTriggerName
}

func (t *OnWebhook) Label() string {
	return "On Workspace Webhook"
}

func (t *OnWebhook) Description() string {
	return "Start when a workspace webhook receives a request"
}

func (t *OnWebhook) Documentation() string {
	return `The On Workspace Webhook trigger starts a workflow when a webhook from the workspace settings receives a request.

Create the webhook in the workspace settings, then select it in a factory automation. More than one automation can use the same webhook.

## Event Data

Each event has type ` + "`webhook`" + `. The payload is:

` + "```" + `
{
  "body": { "...": "the JSON request body" },
  "headers": { "Content-Type": ["application/json"] }
}
` + "```" + `

SuperPlane masks the authentication header before it stores the event.`
}

func (t *OnWebhook) Icon() string {
	return "webhook"
}

func (t *OnWebhook) Color() string {
	return "blue"
}

func (t *OnWebhook) ExampleData() map[string]any {
	return map[string]any{
		"type":      OnWebhookPayloadType,
		"timestamp": "2026-01-01T00:00:00Z",
		"data": map[string]any{
			"body": map[string]any{
				"issue": map[string]any{"key": "ENG-42", "summary": "Checkout fails for saved cards"},
			},
			"headers": map[string]any{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Bearer ********"},
			},
		},
	}
}

func (t *OnWebhook) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "webhook",
			Label:       "Webhook",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "ID of a webhook from the workspace settings",
		},
	}
}

func (t *OnWebhook) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var body any
	if err := models.UnmarshalJSONValue(ctx.Body, &body); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	err := ctx.Events.Emit(OnWebhookPayloadType, map[string]any{
		"body":    body,
		"headers": ctx.Headers,
	})
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnWebhook) Setup(ctx core.TriggerContext) error {
	var config OnWebhookConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	webhookID := strings.TrimSpace(config.Webhook)
	if webhookID == "" {
		return fmt.Errorf("select a webhook")
	}

	binder, ok := ctx.Webhook.(core.WebhookBinder)
	if !ok {
		return fmt.Errorf("this app cannot use workspace webhooks")
	}
	if _, err := binder.Bind(webhookID); err != nil {
		return fmt.Errorf("webhook is not available: %w", err)
	}
	return nil
}

func (t *OnWebhook) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnWebhook) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, fmt.Errorf("hook %s not supported", ctx.Name)
}

func (t *OnWebhook) Cleanup(ctx core.TriggerContext) error {
	return nil
}
