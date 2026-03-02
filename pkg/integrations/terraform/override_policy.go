package terraform

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OverridePolicy struct{}
type OverridePolicySpec struct {
	RunID string `json:"runId"`
}

func (c *OverridePolicy) Name() string        { return "terraform.overridePolicy" }
func (c *OverridePolicy) Label() string       { return "Override Policy" }
func (c *OverridePolicy) Color() string       { return "orange" }
func (c *OverridePolicy) Icon() string        { return "shield" }
func (c *OverridePolicy) Description() string { return "Overrides a failed Sentinel policy block." }
func (c *OverridePolicy) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "runId", Label: "Run ID", Type: configuration.FieldTypeString, Required: true},
	}
}
func (c *OverridePolicy) Execute(ctx core.ExecutionContext) error {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	spec := OverridePolicySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	run, err := client.ReadRun(context.Background(), spec.RunID)
	if err != nil {
		return fmt.Errorf("failed to read run: %w", err)
	}

	if run.Attributes.Status != "policy_override" {
		return fmt.Errorf("state Altered Externally: Run %s is currently '%s', not pending a policy override", spec.RunID, run.Attributes.Status)
	}

	policyChecks, err := client.ListPolicyChecks(context.Background(), spec.RunID)
	if err != nil {
		return fmt.Errorf("failed to list policy checks: %w", err)
	}

	for _, check := range policyChecks.Data {
		if !check.Attributes.Result.Result {
			err = client.OverridePolicy(context.Background(), check.ID)
			if err != nil {
				return fmt.Errorf("failed to override policy check %s: %w", check.ID, err)
			}
		}
	}

	return ctx.ExecutionState.Emit("default", "", []any{map[string]any{"runId": spec.RunID, "action": "overridden"}})
}

func (c *OverridePolicy) Actions() []core.Action                                    { return nil }
func (c *OverridePolicy) HandleAction(ctx core.ActionContext) error                 { return nil }
func (c *OverridePolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return 200, nil }
func (c *OverridePolicy) Cancel(ctx core.ExecutionContext) error                    { return nil }
func (c *OverridePolicy) Setup(ctx core.SetupContext) error                         { return nil }
func (c *OverridePolicy) Cleanup(ctx core.SetupContext) error                       { return nil }
func (c *OverridePolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "default", Label: "Policy Overridden", Description: "Emits when the policy check is successfully overridden"},
	}
}
func (c *OverridePolicy) DefaultOutputChannel() core.OutputChannel { return core.DefaultOutputChannel }
func (c *OverridePolicy) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":  "run-xxxxxx",
		"action": "overridden",
	}
}
func (c *OverridePolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *OverridePolicy) Documentation() string { return "" }
