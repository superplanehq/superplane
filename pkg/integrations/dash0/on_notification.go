package dash0

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnNotification struct{}

type OnNotificationMetadata struct {
	WebhookURL string `json:"webhookUrl" mapstructure:"webhookUrl"`
}

func (t *OnNotification) Name() string {
	return "dash0.onNotification"
}

func (t *OnNotification) Label() string {
	return "On Notification"
}

func (t *OnNotification) Description() string {
	return "Trigger a workflow when Dash0 sends a notification"
}

func (t *OnNotification) Documentation() string {
	return `The On Notification trigger starts a workflow execution when Dash0 sends a notification to SuperPlane.

## Setup

1. Go to the Dash0 integration settings in SuperPlane and copy the webhook URL shown there.
2. In your Dash0 account, configure a notification channel using that URL.
3. When Dash0 sends a notification (e.g. a synthetic check failure), this trigger will fire.

## Output

Emits the raw Dash0 notification payload as ` + "`dash0.notification`" + `.`
}

func (t *OnNotification) Icon() string {
	return "bell"
}

func (t *OnNotification) Color() string {
	return "red"
}

func (t *OnNotification) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnNotification) ExampleData() map[string]any {
	return map[string]any{
		"type":      "synthetic-check-failed",
		"checkName": "Login API health check",
		"severity":  "critical",
		"timestamp": "2026-01-19T12:00:00Z",
	}
}

func (t *OnNotification) Setup(ctx core.TriggerContext) error {
	if err := ctx.Integration.RequestWebhook(struct{}{}); err != nil {
		return err
	}

	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	if ctx.Metadata == nil {
		return nil
	}

	return ctx.Metadata.Set(OnNotificationMetadata{WebhookURL: webhookURL})
}

func (t *OnNotification) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse request body: %w", err)
	}

	if err := ctx.Events.Emit("dash0.notification", payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnNotification) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnNotification) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnNotification) Cleanup(ctx core.TriggerContext) error {
	return nil
}
