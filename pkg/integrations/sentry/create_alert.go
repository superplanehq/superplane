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

type CreateAlert struct{}

type CreateAlertConfiguration struct {
	Project       string                      `json:"project" mapstructure:"project"`
	Name          string                      `json:"name" mapstructure:"name"`
	Aggregate     string                      `json:"aggregate" mapstructure:"aggregate"`
	Query         string                      `json:"query" mapstructure:"query"`
	TimeWindow    float64                     `json:"timeWindow" mapstructure:"timeWindow"`
	ThresholdType string                      `json:"thresholdType" mapstructure:"thresholdType"`
	Environment   string                      `json:"environment" mapstructure:"environment"`
	EventTypes    []string                    `json:"eventTypes" mapstructure:"eventTypes"`
	Critical      AlertThresholdConfiguration `json:"critical" mapstructure:"critical"`
	Warning       AlertThresholdConfiguration `json:"warning" mapstructure:"warning"`
}

func (c *CreateAlert) Name() string {
	return "sentry.createAlert"
}

func (c *CreateAlert) Label() string {
	return "Create Alert"
}

func (c *CreateAlert) Description() string {
	return "Create a Sentry metric alert rule with thresholds, conditions, and notification targets"
}

func (c *CreateAlert) Documentation() string {
	return `The Create Alert component creates a Sentry metric alert rule for a selected project.

## Use Cases

- **Coverage automation**: create alert rules automatically after provisioning a service
- **Policy enforcement**: ensure critical projects always have baseline metric alerts
- **Release safety**: create release-specific alert rules after deploy workflows

## Configuration

- **Project**: Sentry project that owns the metric alert rule
- **Name**: Alert rule name shown in Sentry
- **Aggregate**: Metric expression such as ` + "`count()`" + `
- **Query**: Optional event search query to narrow the alert
- **Time Window**: Evaluation window in minutes
- **Threshold Type**: Whether the threshold fires above or below the configured value
- **Environment**: Optional environment filter
- **Event Types**: Event types included in the alert evaluation
- **Critical / Warning**: Thresholds and notification targets for each trigger level. Select the target type first, then choose a Sentry user or team.

## Output

Returns the created Sentry metric alert rule, including triggers, actions, and project association.`
}

func (c *CreateAlert) Icon() string {
	return "bell"
}

func (c *CreateAlert) Color() string {
	return "gray"
}

func (c *CreateAlert) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateAlert) Configuration() []configuration.Field {
	fields := alertRuleBaseFields(true)
	fields = append(fields,
		alertThresholdField("Critical", "critical", true),
		alertThresholdField("Warning", "warning", false),
	)
	return fields
}

func (c *CreateAlert) Setup(ctx core.SetupContext) error {
	config, err := decodeCreateAlertConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateCreateAlertConfiguration(config); err != nil {
		return err
	}

	project := findProject(ctx.Integration, config.Project)
	if project == nil {
		return fmt.Errorf("project %q was not found in the connected Sentry organization", config.Project)
	}

	return ctx.Metadata.Set(AlertRuleNodeMetadata{
		Project: project,
	})
}

func (c *CreateAlert) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateAlert) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCreateAlertConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateCreateAlertConfiguration(config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	request, err := buildCreateAlertRequest(
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
		return fmt.Errorf("failed to build sentry alert request: %w", err)
	}

	alertRule, err := client.CreateAlertRule(request)
	if err != nil {
		return fmt.Errorf("failed to create sentry alert: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.alertRule", []any{alertRule})
}

func (c *CreateAlert) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateAlert) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateAlert) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateAlert) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeCreateAlertConfiguration(input any) (CreateAlertConfiguration, error) {
	config := CreateAlertConfiguration{}
	if err := mapstructure.Decode(input, &config); err != nil {
		return CreateAlertConfiguration{}, err
	}

	config.Project = strings.TrimSpace(config.Project)
	config.Name = strings.TrimSpace(config.Name)
	config.Aggregate = strings.TrimSpace(config.Aggregate)
	config.Query = strings.TrimSpace(config.Query)
	config.ThresholdType = strings.TrimSpace(config.ThresholdType)
	config.Environment = strings.TrimSpace(config.Environment)
	config.EventTypes = normalizeAlertEventTypes(config.EventTypes)
	config.Critical.Notification.TargetType = strings.TrimSpace(config.Critical.Notification.TargetType)
	config.Critical.Notification.TargetIdentifier = strings.TrimSpace(config.Critical.Notification.TargetIdentifier)
	config.Warning.Notification.TargetType = strings.TrimSpace(config.Warning.Notification.TargetType)
	config.Warning.Notification.TargetIdentifier = strings.TrimSpace(config.Warning.Notification.TargetIdentifier)

	return config, nil
}

func validateCreateAlertConfiguration(config CreateAlertConfiguration) error {
	if config.Project == "" {
		return fmt.Errorf("project is required")
	}
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}
	if config.Aggregate == "" {
		return fmt.Errorf("aggregate is required")
	}
	if config.TimeWindow <= 0 {
		return fmt.Errorf("timeWindow must be greater than 0")
	}

	_, err := buildAlertTriggerInput(alertTriggerLabelCritical, config.Critical)
	if err != nil {
		return err
	}

	if config.Warning.Threshold != nil {
		if strings.TrimSpace(config.Warning.Notification.TargetType) == "" ||
			strings.TrimSpace(config.Warning.Notification.TargetIdentifier) == "" {
			return fmt.Errorf("warning notification target is required when warning threshold is set")
		}

		_, err := buildAlertTriggerInput(alertTriggerLabelWarning, config.Warning)
		if err != nil {
			return err
		}
	}

	return nil
}
