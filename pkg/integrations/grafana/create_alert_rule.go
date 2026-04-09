package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateAlertRule struct{}

func (c *CreateAlertRule) Name() string {
	return "grafana.createAlertRule"
}

func (c *CreateAlertRule) Label() string {
	return "Create Alert Rule"
}

func (c *CreateAlertRule) Description() string {
	return "Create a Grafana-managed alert rule from structured alert settings"
}

func (c *CreateAlertRule) Documentation() string {
	return `The Create Alert Rule component creates a Grafana-managed alert rule using the Alerting Provisioning HTTP API.

## Use Cases

- **Monitoring onboarding**: create baseline alerts when a new service or environment is provisioned
- **Incident automation**: create temporary alert rules during an incident or validation workflow
- **Policy rollout**: standardize alert coverage across teams using a shared rule definition

## Configuration

- **Title**: Human-readable alert name shown in Grafana
- **Folder**: Existing Grafana folder that should contain the rule
- **Rule Group**: Grafana rule group to create the rule in
- **Data Source**: Existing Grafana data source the query should use
- **Query**: Expression Grafana evaluates when checking the alert
- **Lookback Window**: How far back to query when evaluating the rule
- **Reducer / Condition / Threshold(s)**: How the series is reduced, how it is compared to thresholds, and optional upper bound for range conditions
- **For**: How long the condition must hold before firing
- **No Data / Execution Error State**: Grafana behavior when the query returns no data or errors
- **Contact Point**: Optional Grafana contact point for notifications when the rule fires
- **Labels / Annotations**: Optional routing and context metadata attached to the rule
- **Paused**: Whether the rule starts paused

## Output

Returns the created Grafana alert rule object, including identifiers and evaluation metadata.`
}

func (c *CreateAlertRule) Icon() string {
	return "bell"
}

func (c *CreateAlertRule) Color() string {
	return "green"
}

func (c *CreateAlertRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateAlertRule) Configuration() []configuration.Field {
	return alertRuleFieldConfiguration(false, false)
}

func (c *CreateAlertRule) Setup(ctx core.SetupContext) error {
	spec, err := decodeCreateAlertRuleSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateCreateAlertRuleSpec(spec); err != nil {
		return err
	}

	storeAlertRuleNodeMetadata(ctx, "", spec.FolderUID)
	return nil
}

func (c *CreateAlertRule) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCreateAlertRuleSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateCreateAlertRuleSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	rule, err := client.CreateAlertRule(buildAlertRulePayload(spec))
	if err != nil {
		return fmt.Errorf("error creating alert rule: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.alertRule",
		[]any{rule},
	)
}

func (c *CreateAlertRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateAlertRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateAlertRule) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateAlertRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateAlertRule) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateAlertRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
