package digitalocean

import "github.com/superplanehq/superplane/pkg/configuration"

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

// alertPolicyConfigurationFields returns the configuration fields shared between
// Create Alert Policy and Update Alert Policy components.
func alertPolicyConfigurationFields() []configuration.Field {
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
			Description: "Whether the alert policy is active",
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
