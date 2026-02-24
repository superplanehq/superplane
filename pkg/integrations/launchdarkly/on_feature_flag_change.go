package launchdarkly

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

// LaunchDarkly webhook "kind" value for feature flag events.
const KindFlag = "flag"

// LaunchDarkly webhook actions found in the accesses array.
const (
	ActionCreateFlag        = "createFlag"
	ActionUpdateOn          = "updateOn"
	ActionUpdateOffVariation = "updateOffVariation"
	ActionUpdateFallthrough = "updateFallthrough"
	ActionUpdateRules       = "updateRules"
	ActionUpdateTargets     = "updateTargets"
	ActionDeleteFlag        = "deleteFlag"
)

type OnFeatureFlagChange struct{}

type OnFeatureFlagChangeConfiguration struct {
	Events                  []string `json:"events" mapstructure:"events"`
	SigningSecretConfigured bool     `json:"signingSecretConfigured" mapstructure:"signingSecretConfigured"`
}

type OnFeatureFlagChangeMetadata struct {
	WebhookURL              string `json:"webhookUrl" mapstructure:"webhookUrl"`
	SigningSecretConfigured bool   `json:"signingSecretConfigured" mapstructure:"signingSecretConfigured"`
}

func (t *OnFeatureFlagChange) Name() string {
	return "launchdarkly.onFeatureFlagChange"
}

func (t *OnFeatureFlagChange) Label() string {
	return "On Feature Flag Change"
}

func (t *OnFeatureFlagChange) Description() string {
	return "Listen to feature flag change events from LaunchDarkly"
}

func (t *OnFeatureFlagChange) Documentation() string {
	return `The On Feature Flag Change trigger starts a workflow execution when LaunchDarkly sends webhooks for feature flag events.

## Use Cases

- **Deployment automation**: Trigger deployments or rollbacks when feature flags change
- **Audit workflows**: Track and log all feature flag changes for compliance
- **Notification workflows**: Send notifications when flags are created, updated, or deleted
- **Integration workflows**: Sync flag changes with external systems

## Configuration

- **Events**: Select which events to listen for (Flag created, Flag updated, Flag deleted)
- **Webhook signing secret**: Use the **Set signing secret** action below (after creating the webhook in LaunchDarkly) to store the signing secret. It is stored securely and never in the workflow configuration.

## Webhook Setup

After adding this trigger:

1. Save the canvas to generate the webhook URL, then copy it from this panel.
2. In [LaunchDarkly Integrations > Webhooks](https://app.launchdarkly.com/settings/integrations), click **Add webhook**.
3. Paste the webhook URL and optionally provide a name.
4. Check **Sign this webhook** and copy the generated **secret**.
5. Use **Set signing secret** below to store the secret securely.
6. Optionally configure a policy to filter which flag events are sent. For example:
` + "```json" + `
[
  {
    "resources": ["proj/*:env/*:flag/*"],
    "actions": ["*"],
    "effect": "allow"
  }
]
` + "```" + `
7. Save the webhook in LaunchDarkly.`
}

func (t *OnFeatureFlagChange) Icon() string {
	return "launchdarkly"
}

func (t *OnFeatureFlagChange) Color() string {
	return "gray"
}

func (t *OnFeatureFlagChange) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{KindFlag},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Feature flag changes", Value: KindFlag},
					},
				},
			},
		},
	}
}

func (t *OnFeatureFlagChange) Setup(ctx core.TriggerContext) error {
	config := OnFeatureFlagChangeConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event type must be chosen")
	}

	if ctx.Integration == nil {
		return fmt.Errorf("integration is required to set up the LaunchDarkly webhook trigger")
	}

	// Default to flag events if no events configured
	events := config.Events
	if len(events) == 0 {
		events = []string{KindFlag}
	}

	if err := ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: events,
	}); err != nil {
		return err
	}

	var webhookURL string
	if ctx.Webhook != nil {
		var err error
		webhookURL, err = ctx.Webhook.Setup()
		if err != nil {
			return fmt.Errorf("failed to get webhook URL: %w", err)
		}
	}

	if ctx.Metadata != nil {
		metadata := OnFeatureFlagChangeMetadata{}
		_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
		metadata.WebhookURL = webhookURL
		if err := ctx.Metadata.Set(metadata); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	return nil
}

func (t *OnFeatureFlagChange) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "setSecret",
			Description:    "Set or clear the webhook signing secret from your LaunchDarkly webhook",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:        "webhookSigningSecret",
					Label:       "Webhook signing secret",
					Type:        configuration.FieldTypeString,
					Required:    false,
					Sensitive:   true,
					Description: "Paste the signing secret from your LaunchDarkly webhook configuration. Leave empty to clear.",
				},
			},
		},
	}
}

func (t *OnFeatureFlagChange) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name != "setSecret" {
		return nil, fmt.Errorf("action %s not supported", ctx.Name)
	}
	return t.setSecret(ctx)
}

func (t *OnFeatureFlagChange) setSecret(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Webhook == nil {
		return nil, fmt.Errorf("webhook is not available")
	}

	var secret string
	if v, ok := ctx.Parameters["webhookSigningSecret"]; ok && v != nil {
		if s, ok := v.(string); ok {
			secret = strings.TrimSpace(s)
		}
	}

	if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
		return nil, fmt.Errorf("failed to set webhook signing secret: %w", err)
	}

	configured := secret != ""
	if ctx.Metadata != nil {
		metadata := OnFeatureFlagChangeMetadata{}
		_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
		metadata.SigningSecretConfigured = configured
		if err := ctx.Metadata.Set(metadata); err != nil {
			return nil, fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	return map[string]any{"ok": true, "signingSecretConfigured": configured}, nil
}

func (t *OnFeatureFlagChange) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if ctx.Logger != nil {
		ctx.Logger.Infof("launchdarkly webhook: received for workflow %s", ctx.WorkflowID)
	}

	config := OnFeatureFlagChangeConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Verify webhook signature
	signingSecret := resolveSigningSecret(ctx)
	if signingSecret == "" {
		return http.StatusForbidden, fmt.Errorf("signing secret is required for webhook verification; use the Set signing secret action for this trigger")
	}

	signature := ctx.Headers.Get("X-LD-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing X-LD-Signature header")
	}

	if err := crypto.VerifySignature([]byte(signingSecret), ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %w", err)
	}

	// Parse the webhook payload
	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	// LaunchDarkly webhook payloads have a "kind" field (e.g., "flag", "project", "environment")
	// and an "accesses" array with specific actions (e.g., "createFlag", "updateOn", "deleteFlag").
	kind, _ := payload["kind"].(string)
	if kind == "" {
		return http.StatusBadRequest, fmt.Errorf("missing kind in payload")
	}

	// Filter by configured event kinds
	acceptedEvents := config.Events
	if len(acceptedEvents) == 0 {
		acceptedEvents = []string{KindFlag}
	}
	if !slices.Contains(acceptedEvents, kind) {
		if ctx.Logger != nil {
			ctx.Logger.Infof("launchdarkly webhook: event kind %q not in trigger config (configured: %v), acknowledging without emitting", kind, config.Events)
		}
		return http.StatusOK, nil
	}

	// Determine a more specific payload type from the accesses array
	payloadType := "launchdarkly." + kind
	if accesses, ok := payload["accesses"].([]any); ok && len(accesses) > 0 {
		if access, ok := accesses[0].(map[string]any); ok {
			if action, ok := access["action"].(string); ok && action != "" {
				payloadType = "launchdarkly." + kind + "." + action
			}
		}
	}
	if err := ctx.Events.Emit(payloadType, payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %w", err)
	}

	if ctx.Logger != nil {
		ctx.Logger.Infof("launchdarkly webhook: emitted %s for workflow %s", payloadType, ctx.WorkflowID)
	}

	return http.StatusOK, nil
}

func (t *OnFeatureFlagChange) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// resolveSigningSecret returns the webhook signing secret for verification.
func resolveSigningSecret(ctx core.WebhookRequestContext) string {
	if ctx.Webhook == nil {
		return ""
	}
	b, err := ctx.Webhook.GetSecret()
	if err != nil || len(b) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return ""
	}
	return s
}
