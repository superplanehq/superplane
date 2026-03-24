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

type CreateAlertPolicy struct{}

type CreateAlertPolicySpec struct {
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

func (c *CreateAlertPolicy) Name() string {
	return "digitalocean.createAlertPolicy"
}

func (c *CreateAlertPolicy) Label() string {
	return "Create Alert Policy"
}

func (c *CreateAlertPolicy) Description() string {
	return "Create a DigitalOcean monitoring alert policy for droplet metrics"
}

func (c *CreateAlertPolicy) Documentation() string {
	return `The Create Alert Policy component creates a monitoring alert policy that triggers notifications when droplet metrics cross defined thresholds.

> **Note:** Monitoring is only available for droplets that had monitoring enabled during creation. Droplets created without monitoring will not report metrics or trigger alerts.

## Use Cases

- **Capacity management**: Get notified when CPU or memory usage consistently exceeds a safe operating level
- **Performance monitoring**: Detect and respond to high load averages or network saturation
- **Automated workflows**: Chain downstream actions when infrastructure metrics breach limits

## Configuration

- **Description**: Human-readable name for the alert policy (required)
- **Metric Type**: The droplet metric to monitor, such as CPU Usage or Memory Usage (required)
- **Comparison**: Alert when the value is GreaterThan or LessThan the threshold (required)
- **Threshold Value**: The numeric threshold that triggers the alert (required)
- **Evaluation Window**: The rolling time window over which the metric is averaged (required)
- **Droplets**: Specific droplets to scope the policy to (optional)
- **Tags**: Monitor all droplets with matching tags (optional)
- **Enabled**: Whether the alert policy is immediately active (default: true)
- **Email Notifications**: Email addresses to notify when the alert fires (optional)
- **Slack Channel**: Slack channel to post alerts to, e.g. #alerts (optional)
- **Slack Webhook URL**: Incoming webhook URL for the Slack workspace (required when Slack Channel is set)

## Output

Returns the created alert policy including:
- **uuid**: Alert policy UUID for use in Get/Delete operations
- **description**: Human-readable description
- **type**: Metric type being monitored
- **compare**: Comparison operator (GreaterThan/LessThan)
- **value**: Threshold value
- **window**: Evaluation window
- **enabled**: Whether the policy is active
- **alerts**: Configured notification channels (email and/or Slack)

## Important Notes

- At least one notification channel (email or Slack) is required
- **Slack Channel** and **Slack Webhook URL** must be provided together
- Scoping by **Droplets** and **Tags** are independent — you can use either, both, or neither (applies to all droplets)`
}

func (c *CreateAlertPolicy) Icon() string {
	return "bell"
}

func (c *CreateAlertPolicy) Color() string {
	return "orange"
}

func (c *CreateAlertPolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateAlertPolicy) Configuration() []configuration.Field {
	return alertPolicyConfigurationFields()
}

func (c *CreateAlertPolicy) Setup(ctx core.SetupContext) error {
	spec := CreateAlertPolicySpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
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

	if err := resolveAlertPolicyEntitiesMetadata(ctx, spec.Entities); err != nil {
		return fmt.Errorf("error resolving scoped droplets metadata: %v", err)
	}

	return nil
}

func (c *CreateAlertPolicy) Execute(ctx core.ExecutionContext) error {
	spec := CreateAlertPolicySpec{}
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

	req := CreateAlertPolicyRequest{
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

	policy, err := client.CreateAlertPolicy(req)
	if err != nil {
		return fmt.Errorf("failed to create alert policy: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.alertpolicy.created",
		[]any{policy},
	)
}

func (c *CreateAlertPolicy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateAlertPolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateAlertPolicy) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateAlertPolicy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateAlertPolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateAlertPolicy) Cleanup(ctx core.SetupContext) error {
	return nil
}
