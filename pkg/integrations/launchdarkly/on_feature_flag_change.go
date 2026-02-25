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
	ActionCreateFlag         = "createFlag"
	ActionUpdateOn           = "updateOn"
	ActionUpdateOffVariation = "updateOffVariation"
	ActionUpdateFallthrough  = "updateFallthrough"
	ActionUpdateRules        = "updateRules"
	ActionUpdateTargets      = "updateTargets"
	ActionDeleteFlag         = "deleteFlag"
)

type OnFeatureFlagChange struct{}

type OnFeatureFlagChangeConfiguration struct {
	ProjectKey string   `json:"projectKey" mapstructure:"projectKey"`
	FlagKey    string   `json:"flagKey" mapstructure:"flagKey"`
	Actions    []string `json:"actions" mapstructure:"actions"`
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
	return `The On Feature Flag Change trigger starts a workflow execution when LaunchDarkly sends webhooks for a specific feature flag.

## Use Cases

- **Deployment automation**: Trigger deployments or rollbacks when a feature flag changes
- **Audit workflows**: Track and log changes to a specific flag for compliance
- **Notification workflows**: Send notifications when a flag is created, updated, or deleted
- **Integration workflows**: Sync flag changes with external systems

## Configuration

- **Project**: The LaunchDarkly project containing the flag to monitor
- **Feature Flag**: The specific flag to monitor
- **Actions**: Optionally filter by specific actions (e.g. only when a flag is turned on or off). Leave empty to receive all actions.

## Webhook Setup

The webhook is automatically created in LaunchDarkly when you save the canvas. No manual setup is required.

SuperPlane uses the LaunchDarkly API (via your configured API access token) to create a signed webhook scoped to the selected project, and securely stores the auto-generated signing secret. When LaunchDarkly sends events, SuperPlane verifies the signature and filters to the configured flag automatically.`
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
			Name:        "projectKey",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The LaunchDarkly project containing the flag to monitor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "flagKey",
			Label:       "Feature Flag",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The feature flag to monitor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "flag",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "projectKey",
							ValueFrom: &configuration.ParameterValueFrom{Field: "projectKey"},
						},
					},
				},
			},
		},
		{
			Name:        "actions",
			Label:       "Actions",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by specific actions. Leave empty to receive all actions.",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Turned on / off", Value: ActionUpdateOn},
						{Label: "Targeting changed", Value: ActionUpdateTargets},
						{Label: "Rules changed", Value: ActionUpdateRules},
						{Label: "Default rule changed", Value: ActionUpdateFallthrough},
						{Label: "Off variation changed", Value: ActionUpdateOffVariation},
						{Label: "Flag created", Value: ActionCreateFlag},
						{Label: "Flag deleted", Value: ActionDeleteFlag},
					},
				},
			},
		},
	}
}

func (t *OnFeatureFlagChange) Setup(ctx core.TriggerContext) error {
	config := OnFeatureFlagChangeConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.ProjectKey) == "" {
		return fmt.Errorf("project key is required")
	}

	if strings.TrimSpace(config.FlagKey) == "" {
		return fmt.Errorf("flag key is required")
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectKey: config.ProjectKey,
	})
}

func (t *OnFeatureFlagChange) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnFeatureFlagChange) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("action %s not supported", ctx.Name)
}

func (t *OnFeatureFlagChange) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	ctx.Logger.Infof("launchdarkly webhook: received for workflow %s", ctx.WorkflowID)

	config := OnFeatureFlagChangeConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Verify webhook signature
	signingSecret := resolveSigningSecret(ctx)
	if signingSecret == "" {
		return http.StatusForbidden, fmt.Errorf("signing secret is required for webhook verification; the webhook may still be provisioning")
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

	// Only handle flag events
	if kind != KindFlag {
		ctx.Logger.Infof("launchdarkly webhook: event kind %q is not a flag event, acknowledging without emitting", kind)
		return http.StatusOK, nil
	}

	// Extract action and flag key from the accesses array.
	// Resource format: proj/<projKey>:env/<envKey>:flag/<flagKey>
	action := ""
	flagKey := ""
	if accesses, ok := payload["accesses"].([]any); ok && len(accesses) > 0 {
		if access, ok := accesses[0].(map[string]any); ok {
			action, _ = access["action"].(string)
			resource, _ := access["resource"].(string)
			if parts := strings.Split(resource, ":flag/"); len(parts) == 2 {
				flagKey = parts[1]
			}
		}
	}

	// Filter by configured flag key (skip if flag key could not be extracted from payload)
	if config.FlagKey != "" && flagKey != "" && flagKey != config.FlagKey {
		ctx.Logger.Infof("launchdarkly webhook: flag %q does not match configured flag %q, acknowledging without emitting", flagKey, config.FlagKey)
		return http.StatusOK, nil
	}

	// Filter by configured actions (optional — empty means accept all)
	if len(config.Actions) > 0 && !slices.Contains(config.Actions, action) {
		ctx.Logger.Infof("launchdarkly webhook: action %q not in trigger config (configured: %v), acknowledging without emitting", action, config.Actions)
		return http.StatusOK, nil
	}

	// Determine a more specific payload type from the kind and action
	payloadType := "launchdarkly." + kind
	if action != "" {
		payloadType = "launchdarkly." + kind + "." + action
	}

	if err := ctx.Events.Emit(payloadType, payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %w", err)
	}

	ctx.Logger.Infof("launchdarkly webhook: emitted %s for workflow %s", payloadType, ctx.WorkflowID)
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
