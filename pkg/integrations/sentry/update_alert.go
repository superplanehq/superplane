package sentry

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateAlert struct{}

type UpdateAlertConfiguration struct {
	Project       string                      `json:"project" mapstructure:"project"`
	AlertID       string                      `json:"alertId" mapstructure:"alertId"`
	Name          string                      `json:"name" mapstructure:"name"`
	Aggregate     string                      `json:"aggregate" mapstructure:"aggregate"`
	Query         string                      `json:"query" mapstructure:"query"`
	TimeWindow    *float64                    `json:"timeWindow" mapstructure:"timeWindow"`
	ThresholdType string                      `json:"thresholdType" mapstructure:"thresholdType"`
	Environment   string                      `json:"environment" mapstructure:"environment"`
	EventTypes    []string                    `json:"eventTypes" mapstructure:"eventTypes"`
	Critical      AlertThresholdConfiguration `json:"critical" mapstructure:"critical"`
	Warning       AlertThresholdConfiguration `json:"warning" mapstructure:"warning"`
}

func (c *UpdateAlert) Name() string {
	return "sentry.updateAlert"
}

func (c *UpdateAlert) Label() string {
	return "Update Alert"
}

func (c *UpdateAlert) Description() string {
	return "Update a Sentry metric alert rule with new thresholds, conditions, or notification targets"
}

func (c *UpdateAlert) Documentation() string {
	return `The Update Alert component updates an existing Sentry metric alert rule.

## Use Cases

- **Threshold tuning**: adjust thresholds after an incident review
- **Ownership updates**: redirect alert notifications to a different user or team
- **Environment changes**: tighten or loosen alert conditions for a new rollout

## Configuration

- **Project**: Optional project to narrow alert rule selection or replace the rule's project
- **Alert Rule**: Existing Sentry alert rule to update
- **Name / Aggregate / Query / Time Window / Threshold Type / Environment / Event Types**: Optional overrides for the existing rule
- **Critical / Warning**: Optional updates to trigger thresholds and notification targets. Select the target type first, then choose a Sentry user or team.

## Output

Returns the updated Sentry metric alert rule after the change is applied.`
}

func (c *UpdateAlert) Icon() string {
	return "bell"
}

func (c *UpdateAlert) Color() string {
	return "gray"
}

func (c *UpdateAlert) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateAlert) Configuration() []configuration.Field {
	fields := []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional project to narrow alert rule selection",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeProject},
			},
		},
		{
			Name:        "alertId",
			Label:       "Alert Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Sentry metric alert rule to update",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeAlert,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
	}
	baseFields := alertRuleBaseFields(false)
	fields = append(fields, baseFields[1:]...)
	fields = append(fields,
		alertThresholdField("Critical", "critical", false),
		alertThresholdField("Warning", "warning", false),
	)
	return fields
}

func (c *UpdateAlert) Setup(ctx core.SetupContext) error {
	config, err := decodeUpdateAlertConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.AlertID == "" {
		return fmt.Errorf("alertId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	alertRule, err := client.GetAlertRule(config.AlertID)
	if err != nil {
		return fmt.Errorf("failed to retrieve sentry alert: %w", err)
	}

	var project *ProjectSummary
	if config.Project != "" {
		project = findProject(ctx.Integration, config.Project)
		if project == nil {
			return fmt.Errorf("project %q was not found in the connected Sentry organization", config.Project)
		}

		if !alertRuleContainsProject(*alertRule, config.Project) {
			return fmt.Errorf("alert %q is not associated with project %q", alertRule.Name, config.Project)
		}
	}

	return ctx.Metadata.Set(AlertRuleNodeMetadata{
		Project:   project,
		AlertName: displayAlertRuleLabel(*alertRule),
	})
}

func (c *UpdateAlert) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateAlert) Execute(ctx core.ExecutionContext) error {
	config, err := decodeUpdateAlertConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.AlertID == "" {
		return fmt.Errorf("alertId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	alertRule, err := client.GetAlertRule(config.AlertID)
	if err != nil {
		return fmt.Errorf("failed to retrieve sentry alert: %w", err)
	}

	request, err := buildAlertRequestFromRule(
		*alertRule,
		config.Project,
		config.Name,
		config.Aggregate,
		config.Query,
		config.TimeWindow,
		config.ThresholdType,
		config.Environment,
		config.EventTypes,
		config.Critical,
		config.Warning,
	)
	if err != nil {
		return fmt.Errorf("failed to build sentry alert update: %w", err)
	}

	updatedAlert, err := client.UpdateAlertRule(config.AlertID, request)
	if err != nil {
		return fmt.Errorf("failed to update sentry alert: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.alertRule", []any{updatedAlert})
}

func (c *UpdateAlert) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateAlert) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateAlert) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateAlert) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeUpdateAlertConfiguration(input any) (UpdateAlertConfiguration, error) {
	config := UpdateAlertConfiguration{}
	if err := mapstructure.Decode(input, &config); err != nil {
		return UpdateAlertConfiguration{}, err
	}

	config.AlertID = strings.TrimSpace(config.AlertID)
	config.Project = strings.TrimSpace(config.Project)
	config.Name = strings.TrimSpace(config.Name)
	config.Aggregate = strings.TrimSpace(config.Aggregate)
	config.Query = strings.TrimSpace(config.Query)
	config.ThresholdType = strings.TrimSpace(config.ThresholdType)
	config.Environment = strings.TrimSpace(config.Environment)
	config.EventTypes = trimAlertEventTypeSelections(config.EventTypes)
	config.Critical.Notification.TargetType = strings.TrimSpace(config.Critical.Notification.TargetType)
	config.Critical.Notification.TargetIdentifier = strings.TrimSpace(config.Critical.Notification.TargetIdentifier)
	config.Warning.Notification.TargetType = strings.TrimSpace(config.Warning.Notification.TargetType)
	config.Warning.Notification.TargetIdentifier = strings.TrimSpace(config.Warning.Notification.TargetIdentifier)

	return config, nil
}
