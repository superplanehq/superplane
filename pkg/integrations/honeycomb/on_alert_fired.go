package honeycomb

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnAlertFired struct{}

type OnAlertFiredConfiguration struct {
	AlertName string `json:"alertName" mapstructure:"alertName"`
}

type OnAlertFiredMetadata struct {
	WebhookURL   string `json:"webhookUrl" mapstructure:"webhookUrl"`
	SharedSecret string `json:"sharedSecret" mapstructure:"sharedSecret"`
}

func (t *OnAlertFired) Name() string {
	return "honeycomb.onAlertFired"
}

func (t *OnAlertFired) Label() string {
	return "On Alert Fired"
}

func (t *OnAlertFired) Description() string {
	return "Triggers when Honeycomb sends an alert webhook"
}

func (t *OnAlertFired) Icon() string {
	return "honeycomb"
}

func (t *OnAlertFired) Color() string {
	return "yellow"
}

func (t *OnAlertFired) Documentation() string {
	return `
The On Alert Fired trigger starts a workflow execution when Honeycomb sends an alert webhook.

Setup:
1) Add this trigger to a workflow
2) Optionally set Alert Name (used to filter which node fires)
3) SAVE the node
4) Copy Webhook URL and Shared Secret into Honeycomb webhook integration
`
}

func (t *OnAlertFired) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alertName",
			Label:       "Alert Name (optional)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "If set, only webhooks whose payload name matches will trigger this node.",
		},
	}
}

func (t *OnAlertFired) Setup(ctx core.TriggerContext) error {

	if ctx.Webhook == nil {
		if ctx.Integration == nil {
			return nil
		}
		if err := ctx.Integration.RequestWebhook(map[string]any{}); err != nil {
			return fmt.Errorf("failed to request webhook: %w", err)
		}
		return nil
	}

	if ctx.Integration == nil {
		return nil
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {

		if err := ctx.Integration.RequestWebhook(map[string]any{}); err != nil {
			return fmt.Errorf("failed to request webhook: %w", err)
		}

		secret, err = ctx.Webhook.GetSecret()
		if err != nil {
			return fmt.Errorf("failed to get webhook secret after request: %w", err)
		}
	}

	url, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook url: %w", err)
	}

	md := OnAlertFiredMetadata{
		WebhookURL:   strings.TrimSpace(url),
		SharedSecret: strings.TrimSpace(string(secret)),
	}

	if err := ctx.Metadata.Set(md); err != nil {
		return fmt.Errorf("failed to set node metadata: %w", err)
	}

	return nil
}

func (t *OnAlertFired) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAlertFired) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAlertFired) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnAlertFired) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {

	cfg := OnAlertFiredConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	secretBytes, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get webhook secret: %w", err)
	}
	secret := strings.TrimSpace(string(secretBytes))
	if secret == "" {
		return http.StatusInternalServerError, fmt.Errorf("webhook secret is empty")
	}

	provided := strings.TrimSpace(ctx.Headers.Get("X-Honeycomb-Webhook-Token"))

	// Fallbacks
	if provided == "" {
		provided = strings.TrimSpace(ctx.Headers.Get("Authorization"))
	}

	if provided == "" {
		provided = strings.TrimSpace(ctx.Headers.Get("X-Shared-Secret"))
	}

	if provided == "" {
		return http.StatusUnauthorized, fmt.Errorf("missing webhook token")
	}

	if strings.HasPrefix(strings.ToLower(provided), "bearer ") {
		provided = strings.TrimSpace(provided[len("bearer "):])
	}

	providedBytes := []byte(provided)
	secretBytes := []byte(secret)
	if len(providedBytes) != len(secretBytes) {
		subtle.ConstantTimeCompare(secretBytes, secretBytes) // constant-time dummy to avoid leaking length
		return http.StatusForbidden, fmt.Errorf("invalid webhook token")
	}
	if subtle.ConstantTimeCompare(providedBytes, secretBytes) != 1 {
		return http.StatusForbidden, fmt.Errorf("invalid webhook token")
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		payload = map[string]any{"raw": string(ctx.Body)}
	}

	want := strings.TrimSpace(cfg.AlertName)
	if want != "" && !payloadHasAlertName(payload, want) {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit("honeycomb.alert.fired", payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("emit failed: %w", err)
	}

	return http.StatusOK, nil
}

func payloadHasAlertName(payload map[string]any, want string) bool {
	want = strings.TrimSpace(want)

	if name, ok := payload["name"].(string); ok {
		return strings.EqualFold(strings.TrimSpace(name), want)
	}

	if alert, ok := payload["alert"].(map[string]any); ok {
		if name, ok := alert["name"].(string); ok {
			return strings.EqualFold(strings.TrimSpace(name), want)
		}
	}

	return false
}

func (t *OnAlertFired) ExampleData() map[string]any {
	return embeddedExampleDataOnAlertFired()
}
