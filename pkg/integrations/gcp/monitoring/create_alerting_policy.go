package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateAlertingPolicy struct{}

type CreateAlertingPolicySpec struct {
	DisplayName          string   `mapstructure:"displayName"`
	MetricType           string   `mapstructure:"metricType"`
	Comparison           string   `mapstructure:"comparison"`
	Threshold            float64  `mapstructure:"threshold"`
	Duration             string   `mapstructure:"duration"`
	NotificationChannels []string `mapstructure:"notificationChannels"`
	// Enabled is a pointer so an omitted value (nil) can default to true,
	// matching the field default and docs. A plain bool would decode to false
	// when the key is absent (e.g. for programmatic/expression configs).
	Enabled       *bool  `mapstructure:"enabled"`
	Documentation string `mapstructure:"documentation"`
}

func (c *CreateAlertingPolicy) Name() string {
	return "gcp.monitoring.createAlertingPolicy"
}

func (c *CreateAlertingPolicy) Label() string {
	return "Monitoring • Create Alerting Policy"
}

func (c *CreateAlertingPolicy) Description() string {
	return "Create a Cloud Monitoring alerting policy that triggers when a Compute Engine instance metric crosses a threshold"
}

func (c *CreateAlertingPolicy) Documentation() string {
	return `The Create Alerting Policy component creates a Cloud Monitoring alerting policy that fires when a Compute Engine instance metric crosses a threshold for a sustained duration.

## Use Cases

- **Capacity management**: Alert when CPU utilization stays above a safe operating level
- **Performance monitoring**: Detect network or disk throughput saturation
- **Automated workflows**: React to instance metric breaches downstream

## Configuration

- **Display Name**: Human-readable name for the policy (required)
- **Metric**: The Compute Engine instance metric to monitor — CPU utilization, network, or disk (required)
- **Comparison**: Fire when the value is above or below the threshold (required)
- **Threshold**: The numeric threshold (required). For CPU utilization this is a fraction (e.g. ` + "`0.8`" + ` = 80%)
- **Duration**: How long the condition must hold before firing (required)
- **Notification Channels**: Existing Cloud Monitoring notification channels to alert (optional)
- **Enabled**: Whether the policy is active (default: true)
- **Documentation**: Markdown shown in notifications when the policy fires (optional)

## Output

Returns the created policy:
- **name**: Full resource name (` + "`projects/<project>/alertPolicies/<id>`" + `) for use in Get/Update/Delete
- **id**, **displayName**, **enabled**, **combiner**, **conditionsCount**
- **comparison**, **thresholdValue**, **duration**, **filter**: the condition that was created

## Important Notes

- Requires the ` + "`roles/monitoring.editor`" + ` IAM role on the integration's service account
- The policy monitors the metric across **all** Compute Engine instances in the project
- Without notification channels the policy still fires but only surfaces in the Cloud Monitoring console`
}

func (c *CreateAlertingPolicy) Icon() string {
	return "bell"
}

func (c *CreateAlertingPolicy) Color() string {
	return "orange"
}

func (c *CreateAlertingPolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateAlertingPolicy) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Human-readable name for the alerting policy.",
			Placeholder: "e.g. High CPU on production instances",
		},
		{
			Name:        "metricType",
			Label:       "Metric",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The Compute Engine instance metric to monitor.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: metricFieldOptions()},
			},
		},
		{
			Name:        "comparison",
			Label:       "Comparison",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Fire when the metric value is above or below the threshold.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: comparisonOptions},
			},
		},
		{
			Name:        "threshold",
			Label:       "Threshold",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Description: "The numeric threshold that triggers the alert. CPU utilization is a fraction (0.8 = 80%).",
			Placeholder: "e.g. 0.8",
		},
		{
			Name:        "duration",
			Label:       "Duration",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "How long the condition must hold before the policy fires.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: durationOptions},
			},
		},
		{
			Name:        "notificationChannels",
			Label:       "Notification Channels",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Existing notification channels to alert when the policy fires.",
			Placeholder: "Select notification channels",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeNotificationChannel,
					Multi: true,
				},
			},
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Whether the alerting policy is active.",
			Default:     true,
		},
		{
			Name:        "documentation",
			Label:       "Documentation",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Markdown content included in notifications when the policy fires.",
			Placeholder: "e.g. Runbook: scale out the instance group",
		},
	}
}

func (c *CreateAlertingPolicy) Setup(ctx core.SetupContext) error {
	spec := CreateAlertingPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	return validatePolicyCondition(spec.DisplayName, spec.MetricType, spec.Comparison, spec.Duration, true)
}

// validatePolicyCondition validates the shared condition fields. When required
// is false (the update flow) empty values are allowed, but any provided value
// must still be valid.
func validatePolicyCondition(displayName, metricType, comparison, duration string, requireDisplayName bool) error {
	if requireDisplayName && strings.TrimSpace(displayName) == "" {
		return errors.New("displayName is required")
	}
	if metricType == "" {
		return errors.New("metricType is required")
	}
	if _, ok := metricByType(metricType); !ok {
		return fmt.Errorf("invalid metricType %q", metricType)
	}
	if !isValidComparison(comparison) {
		return errors.New("comparison must be COMPARISON_GT or COMPARISON_LT")
	}
	if !isValidDuration(duration) {
		return errors.New("invalid duration: must be one of 0s, 60s, 300s, 600s, 1800s")
	}
	return nil
}

func (c *CreateAlertingPolicy) Execute(ctx core.ExecutionContext) error {
	spec := CreateAlertingPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	if err := validatePolicyCondition(spec.DisplayName, spec.MetricType, spec.Comparison, spec.Duration, true); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	project := client.ProjectID()
	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}
	policy := map[string]any{
		"displayName": strings.TrimSpace(spec.DisplayName),
		"combiner":    "OR",
		"enabled":     enabled,
		"conditions": []any{
			buildThresholdCondition(spec.MetricType, spec.Comparison, spec.Threshold, spec.Duration),
		},
	}
	if len(spec.NotificationChannels) > 0 {
		policy["notificationChannels"] = spec.NotificationChannels
	}
	if doc := strings.TrimSpace(spec.Documentation); doc != "" {
		policy["documentation"] = map[string]any{"content": doc, "mimeType": "text/markdown"}
	}

	endpoint := fmt.Sprintf("%s/projects/%s/alertPolicies", monitoringBaseURL, project)
	body, err := client.PostURL(context.Background(), endpoint, policy)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create alerting policy: %v", err))
	}

	var created alertPolicy
	if err := json.Unmarshal(body, &created); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse alerting policy response: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.monitoring.alertingPolicy.created",
		[]any{policyPayload(&created)},
	)
}

func (c *CreateAlertingPolicy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateAlertingPolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateAlertingPolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateAlertingPolicy) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateAlertingPolicy) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateAlertingPolicy) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
