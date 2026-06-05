package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateAlertingPolicy struct{}

type UpdateAlertingPolicySpec struct {
	AlertPolicy          string   `mapstructure:"alertPolicy"`
	DisplayName          string   `mapstructure:"displayName"`
	MetricType           string   `mapstructure:"metricType"`
	Comparison           string   `mapstructure:"comparison"`
	Threshold            float64  `mapstructure:"threshold"`
	Duration             string   `mapstructure:"duration"`
	Enabled              string   `mapstructure:"enabled"`
	NotificationChannels []string `mapstructure:"notificationChannels"`
	Documentation        string   `mapstructure:"documentation"`
}

func (u *UpdateAlertingPolicy) Name() string {
	return "gcp.monitoring.updateAlertingPolicy"
}

func (u *UpdateAlertingPolicy) Label() string {
	return "Monitoring • Update Alerting Policy"
}

func (u *UpdateAlertingPolicy) Description() string {
	return "Update an existing Cloud Monitoring alerting policy's threshold, state, or notifications"
}

func (u *UpdateAlertingPolicy) Documentation() string {
	return `The Update Alerting Policy component modifies an existing Cloud Monitoring alerting policy in place. Only the fields you set are changed (sent as an update mask).

## Use Cases

- **Threshold tuning**: Adjust the threshold as baselines change
- **Enable/disable**: Toggle a policy on or off during maintenance windows
- **Notification changes**: Re-point a policy at different notification channels

## Configuration

- **Alerting Policy**: The policy to update (required, supports expressions)
- **Display Name**: New name (optional)
- **Metric / Comparison / Threshold / Duration**: Set the **Metric** to rebuild the alert condition. When changing the condition, all four are used together.
- **Enabled**: Enable, disable, or leave unchanged
- **Notification Channels**: Replace the policy's notification channels (optional)
- **Documentation**: New markdown documentation (optional)

## Output

Returns the updated policy:
- **name**, **id**, **displayName**, **enabled**, **combiner**, **conditionsCount**
- **comparison**, **thresholdValue**, **duration**, **filter**: the resulting condition

## Important Notes

- At least one field must be provided
- Changing the **Metric** replaces the policy's condition(s) with a single new threshold condition
- Requires the ` + "`roles/monitoring.editor`" + ` IAM role on the integration's service account`
}

func (u *UpdateAlertingPolicy) Icon() string {
	return "bell"
}

func (u *UpdateAlertingPolicy) Color() string {
	return "orange"
}

func (u *UpdateAlertingPolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateAlertingPolicy) Configuration() []configuration.Field {
	return []configuration.Field{
		alertPolicySelectorField(),
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New human-readable name for the policy.",
			Placeholder: "e.g. High CPU on production instances",
		},
		{
			Name:        "metricType",
			Label:       "Metric",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Set to rebuild the alert condition. Used with Comparison, Threshold, and Duration.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: metricFieldOptions()},
			},
		},
		{
			Name:        "comparison",
			Label:       "Comparison",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Fire when the metric value is above or below the threshold.",
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "metricType", Values: metricValues()},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "metricType", Values: metricValues()},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: comparisonOptions},
			},
		},
		{
			Name:        "threshold",
			Label:       "Threshold",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "The numeric threshold that triggers the alert.",
			Placeholder: "e.g. 0.8",
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "metricType", Values: metricValues()},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "metricType", Values: metricValues()},
			},
		},
		{
			Name:        "duration",
			Label:       "Duration",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "How long the condition must hold before firing.",
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "metricType", Values: metricValues()},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "metricType", Values: metricValues()},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: durationOptions},
			},
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Enable or disable the policy, or leave it unchanged.",
			Default:     "",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Unchanged", Value: ""},
						{Label: "Enabled", Value: "true"},
						{Label: "Disabled", Value: "false"},
					},
				},
			},
		},
		{
			Name:        "notificationChannels",
			Label:       "Notification Channels",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Replace the policy's notification channels.",
			Placeholder: "Select notification channels",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeNotificationChannel,
					Multi: true,
				},
			},
		},
		{
			Name:        "documentation",
			Label:       "Documentation",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New markdown documentation included in notifications.",
			Placeholder: "e.g. Runbook: scale out the instance group",
		},
	}
}

func (u *UpdateAlertingPolicy) Setup(ctx core.SetupContext) error {
	spec := UpdateAlertingPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if err := validateAlertPolicySelection(spec.AlertPolicy); err != nil {
		return err
	}

	if spec.MetricType != "" {
		if err := validatePolicyCondition(spec.DisplayName, spec.MetricType, spec.Comparison, spec.Duration, false); err != nil {
			return err
		}
	}

	if spec.Enabled != "" && spec.Enabled != "true" && spec.Enabled != "false" {
		return errors.New(`enabled must be "true", "false", or empty`)
	}

	if !hasUpdates(spec) {
		return errors.New("at least one field to update is required")
	}

	return nil
}

func hasUpdates(spec UpdateAlertingPolicySpec) bool {
	return strings.TrimSpace(spec.DisplayName) != "" ||
		spec.MetricType != "" ||
		spec.Enabled != "" ||
		len(spec.NotificationChannels) > 0 ||
		strings.TrimSpace(spec.Documentation) != ""
}

func (u *UpdateAlertingPolicy) Execute(ctx core.ExecutionContext) error {
	spec := UpdateAlertingPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	if !hasUpdates(spec) {
		return ctx.ExecutionState.Fail("error", "at least one field to update is required")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	name, err := resolvePolicyName(spec.AlertPolicy, client.ProjectID())
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	policy := map[string]any{}
	var mask []string

	if dn := strings.TrimSpace(spec.DisplayName); dn != "" {
		policy["displayName"] = dn
		mask = append(mask, "displayName")
	}
	if spec.MetricType != "" {
		if err := validatePolicyCondition(spec.DisplayName, spec.MetricType, spec.Comparison, spec.Duration, false); err != nil {
			return ctx.ExecutionState.Fail("error", err.Error())
		}
		policy["conditions"] = []any{
			buildThresholdCondition(spec.MetricType, spec.Comparison, spec.Threshold, spec.Duration),
		}
		mask = append(mask, "conditions")
	}
	if spec.Enabled != "" {
		policy["enabled"] = spec.Enabled == "true"
		mask = append(mask, "enabled")
	}
	if len(spec.NotificationChannels) > 0 {
		policy["notificationChannels"] = spec.NotificationChannels
		mask = append(mask, "notificationChannels")
	}
	if doc := strings.TrimSpace(spec.Documentation); doc != "" {
		policy["documentation"] = map[string]any{"content": doc, "mimeType": "text/markdown"}
		mask = append(mask, "documentation")
	}

	q := url.Values{}
	q.Set("updateMask", strings.Join(mask, ","))
	endpoint := fmt.Sprintf("%s/%s?%s", monitoringBaseURL, name, q.Encode())

	body, err := client.PatchURL(context.Background(), endpoint, policy)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to update alerting policy: %v", err))
	}

	var updated alertPolicy
	if err := json.Unmarshal(body, &updated); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse alerting policy response: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.monitoring.alertingPolicy.updated",
		[]any{policyPayload(&updated)},
	)
}

func (u *UpdateAlertingPolicy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateAlertingPolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateAlertingPolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (u *UpdateAlertingPolicy) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (u *UpdateAlertingPolicy) Hooks() []core.Hook {
	return []core.Hook{}
}

func (u *UpdateAlertingPolicy) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
