package digitalocean

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateAlertPolicy struct{}

type UpdateAlertPolicySpec struct {
	AlertPolicy  string   `json:"alertPolicy" mapstructure:"alertPolicy"`
	Description  string   `json:"description" mapstructure:"description"`
	Type         string   `json:"type" mapstructure:"type"`
	Compare      string   `json:"compare" mapstructure:"compare"`
	Value        float64  `json:"value" mapstructure:"value"`
	Window       string   `json:"window" mapstructure:"window"`
	Entities     []string `json:"entities" mapstructure:"entities"`
	Tags         []string `json:"tags" mapstructure:"tags"`
	Enabled      bool     `json:"enabled" mapstructure:"enabled"`
	Email        []string `json:"email" mapstructure:"email"`
	SlackChannel string   `json:"slackChannel" mapstructure:"slackChannel"`
	SlackURL     string   `json:"slackUrl" mapstructure:"slackUrl"`
}

func (u *UpdateAlertPolicy) Name() string {
	return "digitalocean.updateAlertPolicy"
}

func (u *UpdateAlertPolicy) Label() string {
	return "Update Alert Policy"
}

func (u *UpdateAlertPolicy) Description() string {
	return "Update an existing DigitalOcean monitoring alert policy"
}

func (u *UpdateAlertPolicy) Documentation() string {
	return `The Update Alert Policy component modifies an existing monitoring alert policy with new settings.

> **Note:** Monitoring is only available for droplets that had monitoring enabled during creation. Droplets created without monitoring will not report metrics or trigger alerts.

## Use Cases

- **Threshold tuning**: Adjust alert thresholds in response to changing baselines or scaling events
- **Enable/disable policies**: Toggle alert policies on or off as part of maintenance windows or incident management
- **Notification changes**: Update notification channels (email or Slack) without recreating the policy
- **Automated policy management**: Programmatically adjust alert policies as part of infrastructure workflows

## Configuration

- **Alert Policy**: The alert policy to update (required, supports expressions)
- **Description**: Human-readable name for the alert policy (required)
- **Metric Type**: The droplet metric to monitor, such as CPU Usage or Memory Usage (required)
- **Comparison**: Alert when the value is GreaterThan or LessThan the threshold (required)
- **Threshold Value**: The numeric threshold that triggers the alert (required)
- **Evaluation Window**: The rolling time window over which the metric is averaged (required)
- **Droplets**: Specific droplets to scope the policy to (optional)
- **Tags**: Monitor all droplets with matching tags (optional)
- **Enabled**: Whether the alert policy is active (default: true)
- **Email Notifications**: Email addresses to notify when the alert fires (optional)
- **Slack Channel**: Slack channel to post alerts to, e.g. #alerts (optional)
- **Slack Webhook URL**: Incoming webhook URL for the Slack workspace (required when Slack Channel is set)

## Output

Returns the updated alert policy including:
- **uuid**: Alert policy UUID
- **description**: Human-readable description
- **type**: Metric type being monitored
- **compare**: Comparison operator (GreaterThan/LessThan)
- **value**: Threshold value
- **window**: Evaluation window
- **enabled**: Whether the policy is active
- **alerts**: Configured notification channels (email and/or Slack)

## Important Notes

- The update operation replaces the entire alert policy — all fields must be provided, not just the ones being changed
- At least one notification channel (email or Slack) is required
- **Slack Channel** and **Slack Webhook URL** must be provided together
- Scoping by **Droplets** and **Tags** are independent — you can use either, both, or neither (applies to all droplets)`
}

func (u *UpdateAlertPolicy) Icon() string {
	return "bell"
}

func (u *UpdateAlertPolicy) Color() string {
	return "yellow"
}

func (u *UpdateAlertPolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateAlertPolicy) Configuration() []configuration.Field {
	alertPolicyField := configuration.Field{
		Name:        "alertPolicy",
		Label:       "Alert Policy",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "The alert policy to update",
		Placeholder: "Select alert policy",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           "alert_policy",
				UseNameAsValue: false,
			},
		},
	}

	return append([]configuration.Field{alertPolicyField}, alertPolicyConfigurationFields()...)
}

func (u *UpdateAlertPolicy) Setup(ctx core.SetupContext) error {
	spec := UpdateAlertPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.AlertPolicy == "" {
		return errors.New("alertPolicy is required")
	}

	if spec.Description == "" {
		return errors.New("description is required")
	}

	if spec.Type == "" {
		return errors.New("type is required")
	}

	if spec.Compare == "" {
		return errors.New("compare is required")
	}

	if spec.Window == "" {
		return errors.New("window is required")
	}

	if (spec.SlackChannel != "" && spec.SlackURL == "") || (spec.SlackChannel == "" && spec.SlackURL != "") {
		return errors.New("slackChannel and slackUrl must both be provided together")
	}

	if len(spec.Email) == 0 && spec.SlackChannel == "" {
		return errors.New("at least one notification channel (email or Slack) is required")
	}

	if err := resolveAlertPolicyMetadata(ctx, spec.AlertPolicy); err != nil {
		return fmt.Errorf("error resolving alert policy metadata: %v", err)
	}

	if err := resolveAlertPolicyEntitiesMetadata(ctx, spec.Entities); err != nil {
		return fmt.Errorf("error resolving scoped droplets metadata: %v", err)
	}

	return nil
}

func (u *UpdateAlertPolicy) Execute(ctx core.ExecutionContext) error {
	spec := UpdateAlertPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	alerts := AlertPolicyAlerts{
		Email: spec.Email,
	}

	if spec.SlackChannel != "" && spec.SlackURL != "" {
		alerts.Slack = []AlertPolicySlackDetails{
			{URL: spec.SlackURL, Channel: spec.SlackChannel},
		}
	}

	req := UpdateAlertPolicyRequest{
		Type:        spec.Type,
		Description: spec.Description,
		Compare:     spec.Compare,
		Value:       spec.Value,
		Window:      spec.Window,
		Entities:    spec.Entities,
		Tags:        spec.Tags,
		Enabled:     spec.Enabled,
		Alerts:      alerts,
	}

	policy, err := client.UpdateAlertPolicy(spec.AlertPolicy, req)
	if err != nil {
		return fmt.Errorf("failed to update alert policy: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.alertpolicy.updated",
		[]any{policy},
	)
}

func (u *UpdateAlertPolicy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateAlertPolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateAlertPolicy) Actions() []core.Action {
	return []core.Action{}
}

func (u *UpdateAlertPolicy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (u *UpdateAlertPolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (u *UpdateAlertPolicy) Cleanup(ctx core.SetupContext) error {
	return nil
}
