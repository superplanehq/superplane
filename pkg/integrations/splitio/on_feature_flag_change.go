package splitio

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnFeatureFlagChange struct{}

type OnFeatureFlagChangeConfiguration struct {
	Environments []configuration.Predicate `json:"environments" mapstructure:"environments"`
	Flags        []configuration.Predicate `json:"flags" mapstructure:"flags"`
}

func (t *OnFeatureFlagChange) Name() string {
	return "splitio.onFeatureFlagChange"
}

func (t *OnFeatureFlagChange) Label() string {
	return "On Feature Flag Change"
}

func (t *OnFeatureFlagChange) Description() string {
	return "Listen to feature flag change events from Split.io"
}

func (t *OnFeatureFlagChange) Documentation() string {
	return `The On Feature Flag Change trigger starts a workflow execution when Split.io sends a webhook notification for feature flag changes.

## Use Cases

- **Deployment automation**: Trigger deployments or rollbacks when a feature flag changes
- **Audit workflows**: Track and log changes to flags for compliance
- **Notification workflows**: Send notifications when a flag is modified
- **Integration workflows**: Sync flag changes with external systems

## Configuration

- **Environments**: Optionally filter by environment(s). Leave empty to receive events for all environments.
- **Feature Flags**: Optionally filter by specific flags or patterns. Leave empty to receive events for all flags.

## Webhook Setup

The webhook URL must be manually configured in Split.io:

1. In the Split.io dashboard, go to **Admin settings > Integrations > Marketplace**.
2. Find "Outgoing Webhook" and add a new integration.
3. Paste the webhook URL shown below into the configuration.
4. Select the environments you want to monitor.
5. Save the integration.

Split.io will send a POST request to the webhook URL whenever a feature flag is changed.`
}

func (t *OnFeatureFlagChange) Icon() string {
	return "splitio"
}

func (t *OnFeatureFlagChange) Color() string {
	return "gray"
}

func (t *OnFeatureFlagChange) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "environments",
			Label:       "Environments",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Filter by environment name. Leave empty to receive events for all environments.",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:        "flags",
			Label:       "Feature Flags",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Filter by feature flag name. Leave empty to receive events for all flags.",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

type OnFeatureFlagChangeMetadata struct {
	URL string `json:"url"`
}

func (t *OnFeatureFlagChange) Setup(ctx core.TriggerContext) error {
	var metadata OnFeatureFlagChangeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	if metadata.URL != "" {
		return nil
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	metadata.URL = webhookURL

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

func (t *OnFeatureFlagChange) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnFeatureFlagChange) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("action %s not supported", ctx.Name)
}

func (t *OnFeatureFlagChange) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	ctx.Logger.Infof("splitio webhook: received for workflow %s", ctx.WorkflowID)

	config := OnFeatureFlagChangeConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	// Split.io webhook payloads have a "type" field (should be "split" for feature flags)
	// and a "name" field with the feature flag name.
	eventType, _ := payload["type"].(string)
	if eventType != "split" && eventType != "" {
		ctx.Logger.Infof("splitio webhook: event type %q is not a feature flag event, acknowledging without emitting", eventType)
		return http.StatusOK, nil
	}

	flagName, _ := payload["name"].(string)
	environmentName, _ := payload["environmentName"].(string)

	// Filter by configured environments
	if len(config.Environments) > 0 && environmentName != "" && !configuration.MatchesAnyPredicate(config.Environments, environmentName) {
		ctx.Logger.Infof("splitio webhook: environment %q does not match configured environments, acknowledging without emitting", environmentName)
		return http.StatusOK, nil
	}

	// Filter by configured flags
	if len(config.Flags) > 0 && flagName != "" && !configuration.MatchesAnyPredicate(config.Flags, flagName) {
		ctx.Logger.Infof("splitio webhook: flag %q does not match configured flags, acknowledging without emitting", flagName)
		return http.StatusOK, nil
	}

	payloadType := "splitio.flag"
	if description, _ := payload["description"].(string); description != "" {
		action := inferAction(description)
		if action != "" {
			payloadType = "splitio.flag." + action
		}
	}

	if err := ctx.Events.Emit(payloadType, payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %w", err)
	}

	ctx.Logger.Infof("splitio webhook: emitted %s for workflow %s", payloadType, ctx.WorkflowID)
	return http.StatusOK, nil
}

func (t *OnFeatureFlagChange) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// inferAction attempts to determine the type of change from the description field.
func inferAction(description string) string {
	lower := strings.ToLower(description)
	switch {
	case strings.Contains(lower, "killed"):
		return "killed"
	case strings.Contains(lower, "restored"):
		return "restored"
	case strings.Contains(lower, "created"):
		return "created"
	case strings.Contains(lower, "deleted"):
		return "deleted"
	case strings.Contains(lower, "updated"):
		return "updated"
	default:
		return "changed"
	}
}
