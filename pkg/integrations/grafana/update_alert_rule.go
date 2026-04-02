package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateAlertRule struct{}

func (c *UpdateAlertRule) Name() string {
	return "grafana.updateAlertRule"
}

func (c *UpdateAlertRule) Label() string {
	return "Update Alert Rule"
}

func (c *UpdateAlertRule) Description() string {
	return "Update an existing Grafana-managed alert rule from structured alert settings"
}

func (c *UpdateAlertRule) Documentation() string {
	return `The Update Alert Rule component updates a Grafana-managed alert rule using the Alerting Provisioning HTTP API.

## Use Cases

- **Threshold tuning**: refine alert conditions after incidents or noisy periods
- **Ownership changes**: update labels and annotations used for routing and context
- **Rollout safety**: adjust alert rules during migrations or environment transitions

## Configuration

- **Alert Rule**: The Grafana alert rule UID to update
- **All other fields are optional**: only the values you provide will be changed
- **Folder / Rule Group**: Optional location changes for the rule in Grafana
- **Data Source / Query**: Optional query details Grafana evaluates
- **Lookback / Reducer / Condition / Threshold(s)**: Optional changes to evaluation and thresholds
- **Contact Point**: Set to a contact point to attach notifications; clear the value to remove notification settings from the rule
- **Labels / Annotations**: Optional metadata to update alongside the rule

## Output

Returns the updated Grafana alert rule object after the provisioning API applies the change.`
}

func (c *UpdateAlertRule) Icon() string {
	return "bell"
}

func (c *UpdateAlertRule) Color() string {
	return "blue"
}

func (c *UpdateAlertRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateAlertRule) Configuration() []configuration.Field {
	return alertRuleFieldConfiguration(true, true)
}

func (c *UpdateAlertRule) Setup(ctx core.SetupContext) error {
	spec, err := decodeUpdateAlertRuleSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateUpdateAlertRuleSpec(spec); err != nil {
		return err
	}

	folderUID := ""
	if spec.FolderUID != nil {
		folderUID = *spec.FolderUID
	}

	storeAlertRuleNodeMetadata(ctx, spec.AlertRuleUID, folderUID)
	return nil
}

func (c *UpdateAlertRule) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeUpdateAlertRuleSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateUpdateAlertRuleSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	existingRule, err := client.GetAlertRule(spec.AlertRuleUID)
	if err != nil {
		return fmt.Errorf("error getting existing alert rule: %w", err)
	}
	if err := validateAlertRuleUpdateSupport(existingRule); err != nil {
		return err
	}

	payload, err := mergeAlertRulePayload(existingRule, spec)
	if err != nil {
		return fmt.Errorf("error building updated alert rule payload: %w", err)
	}

	existingProvenance, _ := existingRule["provenance"].(string)
	rule, err := client.UpdateAlertRule(spec.AlertRuleUID, payload, existingProvenance == "")
	if err != nil {
		return fmt.Errorf("error updating alert rule: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.alertRule",
		[]any{rule},
	)
}

func (c *UpdateAlertRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateAlertRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateAlertRule) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateAlertRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateAlertRule) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateAlertRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
