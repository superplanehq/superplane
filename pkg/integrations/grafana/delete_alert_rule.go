package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteAlertRule struct{}

type DeleteAlertRuleOutput struct {
	UID     string `json:"uid" mapstructure:"uid"`
	Title   string `json:"title" mapstructure:"title"`
	Deleted bool   `json:"deleted" mapstructure:"deleted"`
}

func (c *DeleteAlertRule) Name() string {
	return "grafana.deleteAlertRule"
}

func (c *DeleteAlertRule) Label() string {
	return "Delete Alert Rule"
}

func (c *DeleteAlertRule) Description() string {
	return "Delete an existing Grafana-managed alert rule"
}

func (c *DeleteAlertRule) Documentation() string {
	return `The Delete Alert Rule component deletes a Grafana-managed alert rule using the Alerting Provisioning HTTP API.

## Use Cases

- **Alert cleanup**: remove temporary or obsolete rules after a rollout or incident
- **Service retirement**: delete rules that are no longer needed when an environment is decommissioned
- **Controlled cleanup**: pair deletions with approvals, notifications, or audit workflows

## Configuration

- **Alert Rule**: The Grafana alert rule UID to delete

## Output

Returns a confirmation object with the deleted alert rule UID, title, and deletion status.`
}

func (c *DeleteAlertRule) Icon() string {
	return "bell"
}

func (c *DeleteAlertRule) Color() string {
	return "red"
}

func (c *DeleteAlertRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteAlertRule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alertRuleUid",
			Label:       "Alert Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana alert rule to delete",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeAlertRule,
				},
			},
		},
	}
}

func (c *DeleteAlertRule) Setup(ctx core.SetupContext) error {
	spec, err := decodeGetAlertRuleSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateGetAlertRuleSpec(spec); err != nil {
		return err
	}

	storeAlertRuleNodeMetadata(ctx, spec.AlertRuleUID, "")
	return nil
}

func (c *DeleteAlertRule) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetAlertRuleSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateGetAlertRuleSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	rule, err := client.GetAlertRule(spec.AlertRuleUID)
	if err != nil {
		return fmt.Errorf("error getting alert rule: %w", err)
	}

	if err := client.DeleteAlertRule(spec.AlertRuleUID); err != nil {
		return fmt.Errorf("error deleting alert rule: %w", err)
	}

	title, _ := rule["title"].(string)
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.alertRuleDeleted",
		[]any{DeleteAlertRuleOutput{
			UID:     spec.AlertRuleUID,
			Title:   title,
			Deleted: true,
		}},
	)
}

func (c *DeleteAlertRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteAlertRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteAlertRule) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteAlertRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteAlertRule) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteAlertRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
