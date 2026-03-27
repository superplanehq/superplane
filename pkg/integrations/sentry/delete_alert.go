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

type DeleteAlert struct{}

type DeleteAlertConfiguration struct {
	Project string `json:"project" mapstructure:"project"`
	AlertID string `json:"alertId" mapstructure:"alertId"`
}

type DeleteAlertOutput struct {
	ID      string `json:"id" mapstructure:"id"`
	Name    string `json:"name" mapstructure:"name"`
	Deleted bool   `json:"deleted" mapstructure:"deleted"`
}

func (c *DeleteAlert) Name() string {
	return "sentry.deleteAlert"
}

func (c *DeleteAlert) Label() string {
	return "Delete Alert"
}

func (c *DeleteAlert) Description() string {
	return "Delete a Sentry metric alert rule"
}

func (c *DeleteAlert) Documentation() string {
	return `The Delete Alert component deletes an existing Sentry metric alert rule.

## Use Cases

- **Alert cleanup**: remove obsolete rules after a service is retired
- **Policy rotation**: delete temporary alert rules after a rollout is complete
- **CRUD completion**: pair with alert listing and update workflows

## Configuration

- **Project**: Optional project to narrow alert rule selection
- **Alert Rule**: Metric alert rule to delete

## Output

Returns the deleted alert ID and name so downstream steps can record the removal.`
}

func (c *DeleteAlert) Icon() string {
	return "bell"
}

func (c *DeleteAlert) Color() string {
	return "gray"
}

func (c *DeleteAlert) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteAlert) Configuration() []configuration.Field {
	return []configuration.Field{
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
			Description: "Select the Sentry metric alert rule to delete",
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
}

func (c *DeleteAlert) Setup(ctx core.SetupContext) error {
	config, err := decodeDeleteAlertConfiguration(ctx.Configuration)
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

func (c *DeleteAlert) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteAlert) Execute(ctx core.ExecutionContext) error {
	config, err := decodeDeleteAlertConfiguration(ctx.Configuration)
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

	if err := client.DeleteAlertRule(config.AlertID); err != nil {
		return fmt.Errorf("failed to delete sentry alert: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.alertDeleted", []any{
		DeleteAlertOutput{
			ID:      alertRule.ID,
			Name:    alertRule.Name,
			Deleted: true,
		},
	})
}

func (c *DeleteAlert) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteAlert) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteAlert) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteAlert) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeDeleteAlertConfiguration(input any) (DeleteAlertConfiguration, error) {
	config := DeleteAlertConfiguration{}
	if err := mapstructure.Decode(input, &config); err != nil {
		return DeleteAlertConfiguration{}, err
	}

	config.Project = strings.TrimSpace(config.Project)
	config.AlertID = strings.TrimSpace(config.AlertID)
	return config, nil
}
