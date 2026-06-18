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

const maxPolicyConditions = 6

type CreateAlertingPolicy struct{}

type CreateAlertingPolicySpec struct {
	DisplayName           string          `mapstructure:"displayName"`
	PromQL                PromQLSpec      `mapstructure:",squash"`
	Conditions            []ConditionSpec `mapstructure:"conditions"`
	Combiner              string          `mapstructure:"combiner"`
	Severity              string          `mapstructure:"severity"`
	NotificationChannels  []string        `mapstructure:"notificationChannels"`
	UserLabels            []KeyValueSpec  `mapstructure:"userLabels"`
	Enabled               *bool           `mapstructure:"enabled"`
	AutoClose             string          `mapstructure:"autoClose"`
	NotificationRateLimit string          `mapstructure:"notificationRateLimit"`
	Documentation         string          `mapstructure:"documentation"`
	DocumentationSubject  string          `mapstructure:"documentationSubject"`
}

func (c *CreateAlertingPolicy) Name() string {
	return "gcp.monitoring.createAlertingPolicy"
}

func (c *CreateAlertingPolicy) Label() string {
	return "Monitoring • Create Alerting Policy"
}

func (c *CreateAlertingPolicy) Description() string {
	return "Create a Cloud Monitoring alerting policy from an instance-metric threshold or a PromQL query (Managed Prometheus)"
}

func (c *CreateAlertingPolicy) Documentation() string {
	return `The Create Alerting Policy component creates a Cloud Monitoring alerting policy with one or more threshold conditions on Compute Engine instance metrics.

## Use Cases

- **Capacity management**: Alert when CPU stays above a safe level
- **Composite alerts**: Combine multiple conditions (e.g. high CPU AND high network) with a combiner
- **Severity routing**: Tag policies Critical/Error/Warning and rate-limit or auto-close incidents

## Configuration

- **Display Name**: Human-readable name for the policy (required)
- **Condition type**: **Metric threshold** (default) or **PromQL query** (Google Managed Prometheus)
- **Conditions** (threshold type): One or more threshold conditions. Each has:
  - **Metric**, **Comparison** (above ` + "`>`" + ` or below ` + "`<`" + `), **Threshold**, **Duration**
  - Optional **Aligner**, **Rolling window**, **Group reducer** + **Group by fields** (aggregation)
  - Optional **Trigger** by count or percent of time series
- **PromQL query** (PromQL type): a PromQL expression that fires while it returns results, plus an optional **For** duration and **Evaluation interval** — the Prometheus-style alerting rule
- **Combiner**: How multiple conditions combine — OR / AND / AND-with-matching-resource (default OR)
- **Severity**: Critical / Error / Warning (optional)
- **Notification Channels**: Existing channels to alert (optional)
- **User Labels**: Key/value labels on the policy (optional)
- **Enabled**: Whether the policy is active (default: true)
- **Auto-close** / **Notification rate limit**: Alert strategy (optional)
- **Documentation** / **Documentation subject**: Markdown shown in notifications (optional)

## Output

Returns the created policy: **name**, **id**, **displayName**, **enabled**, **combiner**, **severity**, **conditionsCount**, and the first condition's **comparison**, **thresholdValue**, **duration**, **filter**.

## Important Notes

- Requires the ` + "`roles/monitoring.editor`" + ` IAM role on the integration's service account
- Conditions monitor the metric across **all** Compute Engine instances in the project
- Up to 6 conditions per policy`
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
	// Threshold conditions are required only for the threshold kind; the PromQL
	// query is required only for the PromQL kind. Each set is shown for its kind.
	conditions := conditionsField()
	conditions.Required = false
	conditions.RequiredConditions = []configuration.RequiredCondition{{Field: "conditionKind", Values: []string{conditionKindThreshold}}}
	conditions.VisibilityConditions = []configuration.VisibilityCondition{{Field: "conditionKind", Values: []string{conditionKindThreshold}}}

	fields := []configuration.Field{
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Human-readable name for the alerting policy.",
			Placeholder: "e.g. High CPU on production instances",
		},
		conditionKindField(),
		conditions,
	}
	fields = append(fields, promqlConditionFields()...)
	return append(fields, policyOptionFields()...)
}

// conditionsField is the repeatable conditions list, shared by Create and Update.
func conditionsField() configuration.Field {
	maxItems := maxPolicyConditions
	return configuration.Field{
		Name:        "conditions",
		Label:       "Conditions",
		Type:        configuration.FieldTypeList,
		Required:    true,
		Description: "One or more threshold conditions on instance metrics.",
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel:      "Condition",
				MaxItems:       &maxItems,
				ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeObject, Schema: conditionFields()},
			},
		},
	}
}

// policyOptionFields are the policy-level options shared by Create and Update.
func policyOptionFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "combiner",
			Label:       "Condition Combiner",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "How multiple conditions combine to fire the policy.",
			Default:     "OR",
			// A PromQL policy has exactly one condition, so the combiner has no
			// effect there; hide it only for PromQL. The empty value keeps the
			// combiner visible when conditionKind is unset — e.g. on Update, where
			// conditionKind is togglable and usually omitted when conditions are
			// unchanged — so threshold combiners stay editable.
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "conditionKind", Values: []string{conditionKindThreshold, ""}}},
			TypeOptions:          &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: combinerOptions}},
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Incident severity for the policy.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: severityOptions}},
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
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeNotificationChannel, Multi: true},
			},
		},
		{
			Name:        "userLabels",
			Label:       "User Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Key/value labels attached to the policy.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Label",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeObject, Schema: keyValueFields()},
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
			Name:        "autoClose",
			Label:       "Auto-close incidents after",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Automatically close incidents that stop receiving data after this duration.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: autoCloseOptions}},
		},
		{
			Name:        "notificationRateLimit",
			Label:       "Notification rate limit",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Minimum time between repeated notifications for an incident.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: notificationRateLimitOptions}},
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
		{
			Name:        "documentationSubject",
			Label:       "Documentation subject",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Subject line for notifications (overrides the default).",
		},
	}
}

func keyValueFields() []configuration.Field {
	return []configuration.Field{
		{Name: "key", Label: "Key", Type: configuration.FieldTypeString, Required: true},
		{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: false},
	}
}

func validateCreateSpec(spec CreateAlertingPolicySpec) error {
	if strings.TrimSpace(spec.DisplayName) == "" {
		return errors.New("displayName is required")
	}
	if _, err := buildPolicyConditions(spec.PromQL, spec.Conditions); err != nil {
		return err
	}
	if spec.Combiner != "" && !isValidCombiner(spec.Combiner) {
		return errors.New("invalid combiner")
	}
	if !isValidSeverity(spec.Severity) {
		return errors.New("invalid severity")
	}
	return nil
}

func (c *CreateAlertingPolicy) Setup(ctx core.SetupContext) error {
	spec := CreateAlertingPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	return validateCreateSpec(spec)
}

func (c *CreateAlertingPolicy) Execute(ctx core.ExecutionContext) error {
	spec := CreateAlertingPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	if err := validateCreateSpec(spec); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	conditions, err := buildPolicyConditions(spec.PromQL, spec.Conditions)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	combiner := spec.Combiner
	if combiner == "" {
		combiner = "OR"
	}
	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}

	policy := map[string]any{
		"displayName": strings.TrimSpace(spec.DisplayName),
		"combiner":    combiner,
		"enabled":     enabled,
		"conditions":  conditions,
	}
	if spec.Severity != "" {
		policy["severity"] = spec.Severity
	}
	if len(spec.NotificationChannels) > 0 {
		policy["notificationChannels"] = spec.NotificationChannels
	}
	if labels := buildUserLabels(spec.UserLabels); labels != nil {
		policy["userLabels"] = labels
	}
	if doc := buildDocumentation(spec.Documentation, spec.DocumentationSubject); doc != nil {
		policy["documentation"] = doc
	}
	if strategy := buildAlertStrategy(spec.AutoClose, spec.NotificationRateLimit); strategy != nil {
		policy["alertStrategy"] = strategy
	}

	endpoint := fmt.Sprintf("%s/projects/%s/alertPolicies", monitoringBaseURL, client.ProjectID())
	body, err := client.PostURL(context.Background(), endpoint, policy)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to create alerting policy", roleHintWrite, err))
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
