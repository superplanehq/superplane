package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListAlertRules struct{}

type ListAlertRulesOutput struct {
	AlertRules []AlertRuleSummary `json:"alertRules" mapstructure:"alertRules"`
}

func (c *ListAlertRules) Name() string {
	return "grafana.listAlertRules"
}

func (c *ListAlertRules) Label() string {
	return "List Alert Rules"
}

func (c *ListAlertRules) Description() string {
	return "List Grafana-managed alert rules for the connected Grafana instance"
}

func (c *ListAlertRules) Documentation() string {
	return `The List Alert Rules component lists Grafana-managed alert rules using the Alerting Provisioning HTTP API.

## Use Cases

- **Alert audits**: review which Grafana alert rules currently exist
- **Workflow enrichment**: send alert inventories to Slack, Jira, or documentation steps
- **Follow-up automation**: feed alert rule summaries into downstream review or cleanup workflows

## Configuration

This component does not require configuration.

## Output

Returns an object containing the list of Grafana alert rule summaries, including each rule UID and title.`
}

func (c *ListAlertRules) Icon() string {
	return "bell"
}

func (c *ListAlertRules) Color() string {
	return "blue"
}

func (c *ListAlertRules) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListAlertRules) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *ListAlertRules) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *ListAlertRules) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	rules, err := client.ListAlertRules()
	if err != nil {
		return fmt.Errorf("error listing alert rules: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.alertRules",
		[]any{ListAlertRulesOutput{
			AlertRules: rules,
		}},
	)
}

func (c *ListAlertRules) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListAlertRules) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListAlertRules) Actions() []core.Action {
	return []core.Action{}
}

func (c *ListAlertRules) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ListAlertRules) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ListAlertRules) Cleanup(ctx core.SetupContext) error {
	return nil
}
