package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetAlertRule struct{}

func (c *GetAlertRule) Name() string {
	return "grafana.getAlertRule"
}

func (c *GetAlertRule) Label() string {
	return "Get Alert Rule"
}

func (c *GetAlertRule) Description() string {
	return "Retrieve a Grafana-managed alert rule by UID"
}

func (c *GetAlertRule) Documentation() string {
	return `The Get Alert Rule component fetches a Grafana-managed alert rule using the Alerting Provisioning HTTP API.

## Use Cases

- **Configuration review**: inspect the current source of truth before changing a rule
- **Workflow enrichment**: include alert rule details in notifications, tickets, or approvals
- **Drift checks**: compare the current Grafana rule against an expected configuration

## Configuration

- **Alert Rule**: The Grafana alert rule UID to retrieve

## Output

Returns the full Grafana alert rule object, including title, folder, group, condition, queries, labels, and annotations.`
}

func (c *GetAlertRule) Icon() string {
	return "bell"
}

func (c *GetAlertRule) Color() string {
	return "blue"
}

func (c *GetAlertRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetAlertRule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alertRuleUid",
			Label:       "Alert Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana alert rule to retrieve",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeAlertRule,
				},
			},
		},
	}
}

func (c *GetAlertRule) Setup(ctx core.SetupContext) error {
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

func (c *GetAlertRule) Execute(ctx core.ExecutionContext) error {
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

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.alertRule",
		[]any{rule},
	)
}

func (c *GetAlertRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetAlertRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetAlertRule) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetAlertRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetAlertRule) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetAlertRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
