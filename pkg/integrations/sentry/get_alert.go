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

type GetAlert struct{}

type GetAlertConfiguration struct {
	Project string `json:"project" mapstructure:"project"`
	AlertID string `json:"alertId" mapstructure:"alertId"`
}

type GetAlertNodeMetadata struct {
	Project   *ProjectSummary `json:"project,omitempty" mapstructure:"project"`
	AlertName string          `json:"alertName,omitempty" mapstructure:"alertName"`
}

func (c *GetAlert) Name() string {
	return "sentry.getAlert"
}

func (c *GetAlert) Label() string {
	return "Get Alert"
}

func (c *GetAlert) Description() string {
	return "Retrieve a Sentry metric alert rule with its thresholds, projects, and actions"
}

func (c *GetAlert) Documentation() string {
	return `The Get Alert component retrieves a Sentry metric alert rule and returns its full configuration.

## Use Cases

- **Conditional logic**: inspect alert thresholds and projects before taking action
- **Alert enrichment**: include alert configuration in downstream notifications or tickets
- **Auditing**: fetch a specific metric alert rule for verification or reporting

## Configuration

- **Project**: Optional project to narrow alert selection
- **Alert Rule**: The metric alert rule to retrieve

## Output

Returns the selected metric alert rule, including projects, query, thresholds, triggers, and action details.`
}

func (c *GetAlert) Icon() string {
	return "bug"
}

func (c *GetAlert) Color() string {
	return "gray"
}

func (c *GetAlert) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional project to narrow alert rule selection",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "alertId",
			Label:       "Alert Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the metric alert rule to retrieve",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeAlert,
					Parameters: []configuration.ParameterRef{
						{
							Name: "project",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "project",
							},
						},
					},
				},
			},
		},
	}
}

func (c *GetAlert) Setup(ctx core.SetupContext) error {
	config, err := decodeGetAlertConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.AlertID == "" {
		return fmt.Errorf("alertId is required")
	}

	var project *ProjectSummary
	if config.Project != "" {
		project = findProject(ctx.Integration, config.Project)
		if project == nil {
			return fmt.Errorf("project %q was not found in the connected Sentry organization", config.Project)
		}
	}

	if isExpressionValue(config.AlertID) {
		return ctx.Metadata.Set(GetAlertNodeMetadata{
			Project:   project,
			AlertName: "",
		})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	alertRule, err := client.GetAlertRule(config.AlertID)
	if err != nil {
		return fmt.Errorf("failed to retrieve sentry alert: %w", err)
	}

	if config.Project != "" && !alertRuleContainsProject(*alertRule, config.Project) {
		return fmt.Errorf("alert %q is not associated with project %q", alertRule.Name, config.Project)
	}

	return ctx.Metadata.Set(GetAlertNodeMetadata{
		Project:   project,
		AlertName: displayAlertRuleLabel(*alertRule),
	})
}

func (c *GetAlert) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetAlert) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetAlertConfiguration(ctx.Configuration)
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

	if config.Project != "" && !alertRuleContainsProject(*alertRule, config.Project) {
		return fmt.Errorf("alert %q is not associated with project %q", alertRule.Name, config.Project)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.alertRule", []any{alertRule})
}

func (c *GetAlert) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetAlert) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetAlert) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetAlert) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeGetAlertConfiguration(input any) (GetAlertConfiguration, error) {
	config := GetAlertConfiguration{}
	if err := mapstructure.Decode(input, &config); err != nil {
		return GetAlertConfiguration{}, err
	}

	config.Project = strings.TrimSpace(config.Project)
	config.AlertID = strings.TrimSpace(config.AlertID)
	return config, nil
}
