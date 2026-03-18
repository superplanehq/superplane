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

var alertPolicyTypes = []configuration.FieldOption{
	{Label: "CPU Usage (%)", Value: "v1/insights/droplet/cpu"},
	{Label: "Memory Usage (%)", Value: "v1/insights/droplet/memory_utilization_percent"},
	{Label: "Disk Read (bytes/s)", Value: "v1/insights/droplet/disk_read"},
	{Label: "Disk Write (bytes/s)", Value: "v1/insights/droplet/disk_write"},
	{Label: "Public Outbound Bandwidth (Mbps)", Value: "v1/insights/droplet/public_outbound_bandwidth"},
	{Label: "Public Inbound Bandwidth (Mbps)", Value: "v1/insights/droplet/public_inbound_bandwidth"},
	{Label: "Private Outbound Bandwidth (Mbps)", Value: "v1/insights/droplet/private_outbound_bandwidth"},
	{Label: "Private Inbound Bandwidth (Mbps)", Value: "v1/insights/droplet/private_inbound_bandwidth"},
	{Label: "Load Average (1 min)", Value: "v1/insights/droplet/load_1"},
	{Label: "Load Average (5 min)", Value: "v1/insights/droplet/load_5"},
	{Label: "Load Average (15 min)", Value: "v1/insights/droplet/load_15"},
}

var alertPolicyWindows = []configuration.FieldOption{
	{Label: "5 minutes", Value: "5m"},
	{Label: "10 minutes", Value: "10m"},
	{Label: "30 minutes", Value: "30m"},
	{Label: "1 hour", Value: "1h"},
}

var alertPolicyCompare = []configuration.FieldOption{
	{Label: "Greater than", Value: "GreaterThan"},
	{Label: "Less than", Value: "LessThan"},
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

- At least one notification channel (email or Slack) should be configured to actually receive alerts
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
	return []configuration.Field{
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Human-readable name for the alert policy",
			Placeholder: "e.g. High CPU on web servers",
		},
		{
			Name:        "type",
			Label:       "Metric Type",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The droplet metric to monitor",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: alertPolicyTypes,
				},
			},
		},
		{
			Name:        "compare",
			Label:       "Comparison",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Alert when the metric value is greater than or less than the threshold",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: alertPolicyCompare,
				},
			},
		},
		{
			Name:        "value",
			Label:       "Threshold Value",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Description: "The numeric threshold that triggers the alert",
			Placeholder: "e.g. 75",
		},
		{
			Name:        "window",
			Label:       "Evaluation Window",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The rolling time window over which the metric is averaged before comparing to the threshold",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: alertPolicyWindows,
				},
			},
		},
		{
			Name:        "entities",
			Label:       "Droplets",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Specific droplets to apply the alert policy to (optional)",
			Placeholder: "Select droplets",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "droplet",
					Multi: true,
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Monitor all droplets carrying these tags",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Whether the alert policy is immediately active after creation",
			Default:     true,
		},
		{
			Name:        "email",
			Label:       "Email Notifications",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Email addresses to notify when the alert fires",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Email Address",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "slackChannel",
			Label:       "Slack Channel",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Slack channel to notify (e.g. #alerts)",
			Placeholder: "#alerts",
		},
		{
			Name:        "slackUrl",
			Label:       "Slack Webhook URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Incoming webhook URL for the Slack workspace (required when Slack Channel is set)",
			Placeholder: "https://hooks.slack.com/services/...",
		},
	}
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
