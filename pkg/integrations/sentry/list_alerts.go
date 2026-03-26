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

type ListAlerts struct{}

type ListAlertsConfiguration struct {
	Project string `json:"project" mapstructure:"project"`
}

type ListAlertsNodeMetadata struct {
	Project *ProjectSummary `json:"project,omitempty" mapstructure:"project"`
}

type ListAlertsOutput struct {
	Alerts []MetricAlertRule `json:"alerts" mapstructure:"alerts"`
}

func (c *ListAlerts) Name() string {
	return "sentry.listAlerts"
}

func (c *ListAlerts) Label() string {
	return "List Alerts"
}

func (c *ListAlerts) Description() string {
	return "List Sentry metric alert rules for the connected organization or a selected project"
}

func (c *ListAlerts) Documentation() string {
	return `The List Alerts component lists Sentry metric alert rules for the connected organization.

## Use Cases

- **Alert audits**: review metric alert coverage for an organization or project
- **Conditional workflows**: branch based on whether matching alert rules already exist
- **Reporting**: feed alert rule inventories into downstream notification or documentation steps

## Configuration

- **Project**: Optional Sentry project to filter alert rules

## Output

Returns an object containing the list of matching metric alert rules and their configuration details.`
}

func (c *ListAlerts) Icon() string {
	return "bug"
}

func (c *ListAlerts) Color() string {
	return "gray"
}

func (c *ListAlerts) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListAlerts) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional Sentry project to filter alert rules",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
	}
}

func (c *ListAlerts) Setup(ctx core.SetupContext) error {
	config, err := decodeListAlertsConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return ctx.Metadata.Set(ListAlertsNodeMetadata{})
	}

	project := findProject(ctx.Integration, config.Project)
	if project == nil {
		return fmt.Errorf("project %q was not found in the connected Sentry organization", config.Project)
	}

	return ctx.Metadata.Set(ListAlertsNodeMetadata{
		Project: project,
	})
}

func (c *ListAlerts) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListAlerts) Execute(ctx core.ExecutionContext) error {
	config, err := decodeListAlertsConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	alertRules, err := client.ListAlertRules()
	if err != nil {
		return fmt.Errorf("failed to list sentry alerts: %w", err)
	}

	if config.Project != "" {
		filtered := make([]MetricAlertRule, 0, len(alertRules))
		for _, alertRule := range alertRules {
			if alertRuleContainsProject(alertRule, config.Project) {
				filtered = append(filtered, alertRule)
			}
		}
		alertRules = filtered
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.alertRules", []any{
		ListAlertsOutput{Alerts: alertRules},
	})
}

func (c *ListAlerts) Actions() []core.Action {
	return []core.Action{}
}

func (c *ListAlerts) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ListAlerts) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ListAlerts) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListAlerts) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeListAlertsConfiguration(input any) (ListAlertsConfiguration, error) {
	config := ListAlertsConfiguration{}
	if err := mapstructure.Decode(input, &config); err != nil {
		return ListAlertsConfiguration{}, err
	}

	config.Project = strings.TrimSpace(config.Project)
	return config, nil
}
