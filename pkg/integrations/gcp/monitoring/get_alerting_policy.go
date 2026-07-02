package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetAlertingPolicy struct{}

type GetAlertingPolicySpec struct {
	AlertPolicy string `mapstructure:"alertPolicy"`
}

func (g *GetAlertingPolicy) Name() string {
	return "gcp.monitoring.getAlertingPolicy"
}

func (g *GetAlertingPolicy) Label() string {
	return "Monitoring • Get Alerting Policy"
}

func (g *GetAlertingPolicy) Description() string {
	return "Read the configuration and state of a Cloud Monitoring alerting policy"
}

func (g *GetAlertingPolicy) Documentation() string {
	return `The Get Alerting Policy component reads the configuration and state of a Cloud Monitoring alerting policy.

## Use Cases

- **Auditing**: Inspect a policy's threshold, comparison, and enabled state
- **Conditional workflows**: Branch on whether a policy is enabled or how it's configured
- **Chaining**: Read a policy created upstream before updating or deleting it

## Configuration

- **Alerting Policy**: Pick from the policies in your project, or pass an expression chained from an upstream node (e.g. the ` + "`name`" + ` emitted by ` + "`gcp.monitoring.createAlertingPolicy`" + `).

## Output

Returns the policy:
- **name**, **id**, **displayName**, **enabled**, **combiner**, **conditionsCount**
- **comparison**, **thresholdValue**, **duration**, **filter**: the first condition's threshold
- **notificationChannels**: attached channels (when any)

## Important Notes

- Requires the ` + "`roles/monitoring.viewer`" + ` IAM role on the integration's service account
- If the policy is not found, the action fails so stale expressions don't silently mask a problem`
}

func (g *GetAlertingPolicy) Icon() string {
	return "bell"
}

func (g *GetAlertingPolicy) Color() string {
	return "blue"
}

func (g *GetAlertingPolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetAlertingPolicy) Configuration() []configuration.Field {
	return []configuration.Field{alertPolicySelectorField()}
}

func (g *GetAlertingPolicy) Setup(ctx core.SetupContext) error {
	spec := GetAlertingPolicySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if err := validateAlertPolicySelection(spec.AlertPolicy); err != nil {
		return err
	}
	return resolveAlertPolicyMetadata(ctx, spec.AlertPolicy)
}

func (g *GetAlertingPolicy) Execute(ctx core.ExecutionContext) error {
	spec := GetAlertingPolicySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	name, err := resolvePolicyName(spec.AlertPolicy, client.ProjectID())
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	body, err := client.GetURL(context.Background(), fmt.Sprintf("%s/%s", monitoringBaseURL, name))
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to get alerting policy", roleHintRead, err))
	}

	var policy alertPolicy
	if err := json.Unmarshal(body, &policy); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("parse alerting policy response: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.monitoring.alertingPolicy.fetched",
		[]any{policyPayload(&policy)},
	)
}

func (g *GetAlertingPolicy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetAlertingPolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetAlertingPolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetAlertingPolicy) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetAlertingPolicy) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetAlertingPolicy) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

// alertPolicySelectorField is the shared "pick an alert policy" field used by the
// Get, Delete, and Update components.
func alertPolicySelectorField() configuration.Field {
	return configuration.Field{
		Name:        "alertPolicy",
		Label:       "Alerting Policy",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "The alerting policy to target. Lists the policies in your project.",
		Placeholder: "Select alerting policy",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: ResourceTypeAlertPolicy,
			},
		},
	}
}

func validateAlertPolicySelection(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("alertPolicy is required")
	}
	// Expressions are resolved at execution time.
	if strings.Contains(value, "{{") {
		return nil
	}
	_, _, err := parsePolicyName(value)
	return err
}

// resolvePolicyName parses the selected value into a relative policy name and
// verifies it belongs to the integration's project.
func resolvePolicyName(value, project string) (string, error) {
	urlProject, name, err := parsePolicyName(value)
	if err != nil {
		return "", err
	}
	if urlProject != project {
		return "", fmt.Errorf(
			"alert policy belongs to project %q but this GCP integration is bound to project %q; cross-project operations are not supported",
			urlProject, project,
		)
	}
	return name, nil
}

// AlertPolicyNodeMetadata is stored on the node at Setup so the collapsed UI can
// show the policy's human-readable display name instead of its numeric ID.
type AlertPolicyNodeMetadata struct {
	PolicyName  string `json:"policyName" mapstructure:"policyName"`
	DisplayName string `json:"displayName" mapstructure:"displayName"`
	ID          string `json:"id" mapstructure:"id"`
}

// resolveAlertPolicyMetadata best-effort resolves the selected policy's display
// name via the API and stores it on the node. It falls back to the parsed ID
// when the value is an expression or the API is unavailable, so Setup never
// fails just because the display name could not be fetched.
func resolveAlertPolicyMetadata(ctx core.SetupContext, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	// Expressions are resolved at execution time; store the raw value so the UI
	// still shows something meaningful.
	if strings.Contains(value, "{{") {
		return ctx.Metadata.Set(AlertPolicyNodeMetadata{PolicyName: value})
	}

	urlProject, name, err := parsePolicyName(value)
	if err != nil {
		return err
	}

	fallback := AlertPolicyNodeMetadata{PolicyName: name, ID: lastSegment(name)}
	if ctx.Integration == nil {
		return ctx.Metadata.Set(fallback)
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.Metadata.Set(fallback)
	}
	if urlProject != "" && client.ProjectID() != "" && urlProject != client.ProjectID() {
		return ctx.Metadata.Set(fallback)
	}

	body, err := client.GetURL(context.Background(), fmt.Sprintf("%s/%s", monitoringBaseURL, name))
	if err != nil {
		return ctx.Metadata.Set(fallback)
	}
	var p alertPolicy
	if err := json.Unmarshal(body, &p); err != nil {
		return ctx.Metadata.Set(fallback)
	}
	display := p.DisplayName
	if display == "" {
		display = lastSegment(name)
	}
	return ctx.Metadata.Set(AlertPolicyNodeMetadata{
		PolicyName:  name,
		DisplayName: display,
		ID:          lastSegment(name),
	})
}
