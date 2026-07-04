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
	AlertPolicy           string          `mapstructure:"alertPolicy"`
	DisplayName           string          `mapstructure:"displayName"`
	PromQL                PromQLSpec      `mapstructure:",squash"`
	Conditions            []ConditionSpec `mapstructure:"conditions"`
	Combiner              string          `mapstructure:"combiner"`
	Severity              string          `mapstructure:"severity"`
	NotificationChannels  []string        `mapstructure:"notificationChannels"`
	UserLabels            []KeyValueSpec  `mapstructure:"userLabels"`
	Enabled               string          `mapstructure:"enabled"`
	AutoClose             string          `mapstructure:"autoClose"`
	NotificationRateLimit string          `mapstructure:"notificationRateLimit"`
	Documentation         string          `mapstructure:"documentation"`
	DocumentationSubject  string          `mapstructure:"documentationSubject"`
}

func (u *UpdateAlertingPolicy) Name() string {
	return "gcp.monitoring.updateAlertingPolicy"
}

func (u *UpdateAlertingPolicy) Label() string {
	return "Monitoring • Update Alerting Policy"
}

func (u *UpdateAlertingPolicy) Description() string {
	return "Update an existing Cloud Monitoring alerting policy's conditions, combiner, severity, strategy, or notifications"
}

func (u *UpdateAlertingPolicy) Documentation() string {
	return `The Update Alerting Policy component modifies an existing Cloud Monitoring alerting policy in place. Only the fields you set are changed (sent as an update mask).

## Use Cases

- **Threshold tuning**: Adjust a condition's threshold as baselines change
- **Enable/disable**: Toggle a policy during maintenance windows
- **Re-route**: Change notification channels, severity, or alert strategy

## Configuration

- **Alerting Policy**: The policy to update (required, supports expressions)
- **Condition type**: Toggle to **replace** the policy's conditions with a metric threshold or a PromQL query (leave off to keep current conditions)
- **Conditions** (threshold type): Provide to replace with threshold conditions (each: metric, comparison, threshold, duration, optional aggregation/trigger)
- **PromQL query** (PromQL type): Provide to replace with a single PromQL condition (with optional For duration and evaluation interval)
- **Combiner**: OR / AND / AND-with-matching-resource
- **Severity**: Critical / Error / Warning
- **Enabled**: Enable, disable, or leave unchanged
- **Notification Channels**: Replace channels (provide empty to clear)
- **User Labels**: Replace user labels
- **Auto-close / Notification rate limit**: Replace the alert strategy
- **Documentation / subject**: Replace the documentation

## Output

Returns the updated policy: **name**, **id**, **displayName**, **enabled**, **combiner**, **severity**, **conditionsCount**, and the first condition summary.

## Important Notes

- At least one field must be provided
- Providing **Conditions** replaces all existing conditions
- **Auto-close**, **rate limit**, **documentation content**, and **documentation subject** are each updated independently — changing one leaves the others untouched
- Requires the ` + "`roles/monitoring.editor`" + ` IAM role`
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
	fields := []configuration.Field{alertPolicySelectorField()}

	// Conditions are optional on update (provide to replace them). The condition
	// kind toggle and PromQL inputs are togglable too, so a PromQL query can
	// replace the policy's conditions.
	conditionKind := conditionKindField()
	conditionKind.Togglable = true
	conditionKind.Description = "Replace the policy's conditions with a metric threshold or a PromQL query (leave off to keep current conditions)."
	fields = append(fields, conditionKind)

	conditions := conditionsField()
	conditions.Required = false
	conditions.Togglable = true
	conditions.VisibilityConditions = []configuration.VisibilityCondition{{Field: "conditionKind", Values: []string{conditionKindThreshold}}}
	fields = append(fields, conditions)

	for _, f := range promqlConditionFields() {
		f.Togglable = true
		f.RequiredConditions = nil
		fields = append(fields, f)
	}

	// Reuse the create option fields, but make combiner/enabled "unchanged"
	// friendly: no combiner default, and enabled becomes a 3-way select.
	for _, f := range policyOptionFields() {
		switch f.Name {
		case "combiner":
			f.Default = nil
			f.Description = "Change how conditions combine (leave unset to keep current)."
		case "enabled":
			f.Type = configuration.FieldTypeSelect
			f.Togglable = true
			f.Default = nil
			f.Description = "Enable or disable the policy (leave off to keep unchanged)."
			f.TypeOptions = &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Enabled", Value: "true"},
						{Label: "Disabled", Value: "false"},
					},
				},
			}
		}
		fields = append(fields, f)
	}
	return fields
}

func (u *UpdateAlertingPolicy) Setup(ctx core.SetupContext) error {
	spec := UpdateAlertingPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if err := validateAlertPolicySelection(spec.AlertPolicy); err != nil {
		return err
	}
	if err := validateUpdateFields(spec, ctx.Configuration); err != nil {
		return err
	}
	if !hasUpdates(spec, ctx.Configuration) {
		return errors.New("at least one field to update is required")
	}

	return resolveAlertPolicyMetadata(ctx, spec.AlertPolicy)
}

// isPromQLConditionUpdate reports whether the spec asks to replace the policy's
// conditions with a PromQL condition. A saved conditionKind of "promql" alone is
// not enough — the query must also be present — so that an update toggled to
// PromQL but left without a query does not block unrelated field updates (e.g.
// severity or display name), mirroring how the threshold path only acts when its
// "conditions" key is present.
func isPromQLConditionUpdate(spec UpdateAlertingPolicySpec) bool {
	return spec.PromQL.ConditionKind == conditionKindPromQL && strings.TrimSpace(spec.PromQL.Query) != ""
}

func validateUpdateFields(spec UpdateAlertingPolicySpec, cfg any) error {
	switch {
	case isPromQLConditionUpdate(spec):
		if _, err := buildPromQLConditions(spec.PromQL); err != nil {
			return err
		}
	case configHasKey(cfg, "conditions"):
		if _, err := buildConditions(spec.Conditions); err != nil {
			return err
		}
	}
	if spec.Combiner != "" && !isValidCombiner(spec.Combiner) {
		return errors.New("invalid combiner")
	}
	if !isValidSeverity(spec.Severity) {
		return errors.New("invalid severity")
	}
	return validateEnabledOption(spec.Enabled)
}

// validateEnabledOption guards the three-way enabled select. It is enforced in
// both Setup and Execute so a mistyped value surfaces an error rather than
// silently disabling the policy.
func validateEnabledOption(enabled string) error {
	if enabled != "" && enabled != "true" && enabled != "false" {
		return errors.New(`enabled must be "true", "false", or empty`)
	}
	return nil
}

func configHasKey(cfg any, key string) bool {
	m, ok := cfg.(map[string]any)
	if !ok {
		return false
	}
	_, ok = m[key]
	return ok
}

// hasUpdates reports whether the spec carries at least one change. Togglable
// fields are tracked by key presence so empty values can be sent to clear them.
func hasUpdates(spec UpdateAlertingPolicySpec, cfg any) bool {
	return strings.TrimSpace(spec.DisplayName) != "" ||
		configHasKey(cfg, "conditions") ||
		isPromQLConditionUpdate(spec) ||
		spec.Combiner != "" ||
		spec.Severity != "" ||
		spec.Enabled != "" ||
		configHasKey(cfg, "notificationChannels") ||
		configHasKey(cfg, "userLabels") ||
		configHasKey(cfg, "autoClose") ||
		configHasKey(cfg, "notificationRateLimit") ||
		configHasKey(cfg, "documentation") ||
		configHasKey(cfg, "documentationSubject")
}

// buildStrategyUpdate assembles the alertStrategy patch body and its field-mask
// paths. Each sub-field is masked independently so changing only one (e.g.
// auto-close) does not clear the sibling (e.g. notification rate limit). A field
// whose key is present but empty is omitted from the body, which clears it under
// its own mask path.
func buildStrategyUpdate(spec UpdateAlertingPolicySpec, cfg any) (map[string]any, []string) {
	strategy := map[string]any{}
	var paths []string
	if configHasKey(cfg, "autoClose") {
		if spec.AutoClose != "" {
			strategy["autoClose"] = spec.AutoClose
		}
		paths = append(paths, "alertStrategy.autoClose")
	}
	if configHasKey(cfg, "notificationRateLimit") {
		if spec.NotificationRateLimit != "" {
			strategy["notificationRateLimit"] = map[string]any{"period": spec.NotificationRateLimit}
		}
		paths = append(paths, "alertStrategy.notificationRateLimit")
	}
	return strategy, paths
}

// buildDocumentationUpdate assembles the documentation patch body and its
// field-mask paths, masking content and subject independently so updating one
// leaves the other intact.
func buildDocumentationUpdate(spec UpdateAlertingPolicySpec, cfg any) (map[string]any, []string) {
	doc := map[string]any{}
	var paths []string
	if configHasKey(cfg, "documentation") {
		if content := strings.TrimSpace(spec.Documentation); content != "" {
			doc["content"] = content
			doc["mimeType"] = "text/markdown"
		}
		paths = append(paths, "documentation.content")
	}
	if configHasKey(cfg, "documentationSubject") {
		if subject := strings.TrimSpace(spec.DocumentationSubject); subject != "" {
			doc["subject"] = subject
		}
		paths = append(paths, "documentation.subject")
	}
	return doc, paths
}

func (u *UpdateAlertingPolicy) Execute(ctx core.ExecutionContext) error {
	spec := UpdateAlertingPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	if !hasUpdates(spec, ctx.Configuration) {
		return ctx.ExecutionState.Fail("error", "at least one field to update is required")
	}
	if err := validateUpdateFields(spec, ctx.Configuration); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
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
	switch {
	case isPromQLConditionUpdate(spec):
		conditions, err := buildPromQLConditions(spec.PromQL)
		if err != nil {
			return ctx.ExecutionState.Fail("error", err.Error())
		}
		policy["conditions"] = conditions
		mask = append(mask, "conditions")
	case configHasKey(ctx.Configuration, "conditions"):
		conditions, err := buildConditions(spec.Conditions)
		if err != nil {
			return ctx.ExecutionState.Fail("error", err.Error())
		}
		policy["conditions"] = conditions
		mask = append(mask, "conditions")
	}
	if spec.Combiner != "" {
		policy["combiner"] = spec.Combiner
		mask = append(mask, "combiner")
	}
	if spec.Severity != "" {
		policy["severity"] = spec.Severity
		mask = append(mask, "severity")
	}
	if spec.Enabled != "" {
		policy["enabled"] = spec.Enabled == "true"
		mask = append(mask, "enabled")
	}
	if configHasKey(ctx.Configuration, "notificationChannels") {
		channels := spec.NotificationChannels
		if channels == nil {
			channels = []string{}
		}
		policy["notificationChannels"] = channels
		mask = append(mask, "notificationChannels")
	}
	if configHasKey(ctx.Configuration, "userLabels") {
		labels := buildUserLabels(spec.UserLabels)
		if labels == nil {
			labels = map[string]string{}
		}
		policy["userLabels"] = labels
		mask = append(mask, "userLabels")
	}
	if strategy, paths := buildStrategyUpdate(spec, ctx.Configuration); len(paths) > 0 {
		policy["alertStrategy"] = strategy
		mask = append(mask, paths...)
	}
	if doc, paths := buildDocumentationUpdate(spec, ctx.Configuration); len(paths) > 0 {
		policy["documentation"] = doc
		mask = append(mask, paths...)
	}

	q := url.Values{}
	q.Set("updateMask", strings.Join(mask, ","))
	endpoint := fmt.Sprintf("%s/%s?%s", monitoringBaseURL, name, q.Encode())

	body, err := client.PatchURL(context.Background(), endpoint, policy)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to update alerting policy", roleHintWrite, err))
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
