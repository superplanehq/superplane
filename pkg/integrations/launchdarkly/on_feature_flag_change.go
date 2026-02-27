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
	ProjectKey   string                    `json:"projectKey" mapstructure:"projectKey"`
	Environments []string                  `json:"environments" mapstructure:"environments"`
	Flags        []configuration.Predicate `json:"flags" mapstructure:"flags"`
	Actions      []string                  `json:"actions" mapstructure:"actions"`
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
	return `The On Feature Flag Change trigger starts a workflow execution when LaunchDarkly sends webhooks for feature flags in a project.

## Use Cases

- **Deployment automation**: Trigger deployments or rollbacks when a feature flag changes
- **Audit workflows**: Track and log changes to flags for compliance
- **Notification workflows**: Send notifications when a flag is created, updated, or deleted
- **Integration workflows**: Sync flag changes with external systems

## Configuration

- **Project**: The LaunchDarkly project to monitor
- **Environments**: Optionally filter by environment(s). Leave empty to receive events for all environments.
- **Feature Flags**: Optionally filter by specific flags or patterns. Leave empty to receive events for all flags.
- **Actions**: Optionally filter by specific actions (e.g. only when a flag is turned on or off). Leave empty to receive all actions.

## Webhook Setup

The webhook is automatically created in LaunchDarkly when you save the canvas. No manual setup is required.

SuperPlane uses the LaunchDarkly API (via your configured API access token) to create a signed webhook scoped to the selected project, and securely stores the auto-generated signing secret. When LaunchDarkly sends events, SuperPlane verifies the signature and filters to the configured environments, flags, and actions automatically.`
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
			Description: "The LaunchDarkly project to monitor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "environments",
			Label:       "Environments",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter by environment. Leave empty to receive events for all environments.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "environment",
					Multi: true,
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
			Name:        "flags",
			Label:       "Feature Flags",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Filter by feature flag. Leave empty to receive events for all flags.",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
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

	// Extract action, environment key, and flag key from the accesses array.
	// Resource format: proj/<projKey>:env/<envKey>:flag/<flagKey>
	action := ""
	envKey := ""
	flagKey := ""
	if accesses, ok := payload["accesses"].([]any); ok && len(accesses) > 0 {
		if access, ok := accesses[0].(map[string]any); ok {
			action, _ = access["action"].(string)
			resource, _ := access["resource"].(string)
			envKey, flagKey = parseResourceEnvAndFlag(resource)
		}
	}

	// Filter by configured environments.
	// Skip if: env key could not be extracted (no accesses), or env is "*" (project-scoped
	// actions like createFlag use proj/<proj>:env/*:flag/<flag> and are not environment-specific).
	if len(config.Environments) > 0 && envKey != "" && envKey != "*" && !slices.Contains(config.Environments, envKey) {
		ctx.Logger.Infof("launchdarkly webhook: environment %q does not match configured environments, acknowledging without emitting", envKey)
		return http.StatusOK, nil
	}

	// Filter by configured flags.
	// Skip if: flag key could not be extracted (no accesses).
	if len(config.Flags) > 0 && flagKey != "" && !configuration.MatchesAnyPredicate(config.Flags, flagKey) {
		ctx.Logger.Infof("launchdarkly webhook: flag %q does not match configured flags, acknowledging without emitting", flagKey)
		return http.StatusOK, nil
	}

	// Filter by configured actions (optional — empty means accept all)
	if len(config.Actions) > 0 && !slices.Contains(config.Actions, action) {
		ctx.Logger.Infof("launchdarkly webhook: action %q not in trigger config (configured: %v), acknowledging without emitting", action, config.Actions)
		return http.StatusOK, nil
	}

	// Inject extracted keys into the payload so consumers can access them directly.
	payload["projectKey"] = config.ProjectKey
	if envKey != "" && envKey != "*" {
		payload["environmentKey"] = envKey
	}
	if flagKey != "" {
		payload["flagKey"] = flagKey
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

// parseResourceEnvAndFlag extracts the environment and flag keys from a LaunchDarkly resource string.
// Expected format: proj/<projKey>:env/<envKey>:flag/<flagKey>
func parseResourceEnvAndFlag(resource string) (envKey, flagKey string) {
	// Split on ":env/" to get the environment and flag parts
	envParts := strings.SplitN(resource, ":env/", 2)
	if len(envParts) != 2 {
		return "", ""
	}
	// The remaining part is "<envKey>:flag/<flagKey>"
	flagParts := strings.SplitN(envParts[1], ":flag/", 2)
	if len(flagParts) != 2 {
		return envParts[1], ""
	}
	return flagParts[0], flagParts[1]
}

// resolveSigningSecret returns the webhook signing secret for verification.
func resolveSigningSecret(ctx core.WebhookRequestContext) string {
	b, err := ctx.Webhook.GetSecret()
	if err != nil || len(b) == 0 {
		return ""
	}
	return strings.TrimSpace(string(b))
}
