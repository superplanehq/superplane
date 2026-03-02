package terraform

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ApplyRun struct{}
type ApplyRunSpec struct {
	RunID   string `json:"runId"`
	Comment string `json:"comment"`
}

func (c *ApplyRun) Name() string  { return "terraform.applyRun" }
func (c *ApplyRun) Label() string { return "Apply Run" }
func (c *ApplyRun) Description() string {
	return "Applies a run that is paused in 'needs attention' or 'planned'."
}
func (c *ApplyRun) Icon() string  { return "check-circle" }
func (c *ApplyRun) Color() string { return "green" }
func (c *ApplyRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "runId", Label: "Run ID", Type: configuration.FieldTypeString, Required: true},
		{Name: "comment", Label: "Comment", Type: configuration.FieldTypeString, Required: false},
	}
}
func (c *ApplyRun) Setup(ctx core.SetupContext) error { return nil }
func (c *ApplyRun) Execute(ctx core.ExecutionContext) error {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	spec := ApplyRunSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	run, err := client.ReadRun(context.Background(), spec.RunID)
	if err != nil {
		return fmt.Errorf("failed to read run: %w", err)
	}

	confirmableStates := map[string]bool{
		"planned":        true,
		"cost_estimated": true,
		"policy_checked": true,
	}
	if !confirmableStates[run.Attributes.Status] {
		return fmt.Errorf("run %s is currently '%s', cannot apply (must be 'planned', 'cost_estimated', or 'policy_checked')", spec.RunID, run.Attributes.Status)
	}

	err = client.ApplyRun(context.Background(), spec.RunID, spec.Comment)
	if err != nil {
		return fmt.Errorf("failed to apply run: %w", err)
	}
	return ctx.ExecutionState.Emit("default", "", []any{map[string]any{"runId": spec.RunID, "action": "applied"}})
}

func (c *ApplyRun) Actions() []core.Action                                    { return nil }
func (c *ApplyRun) HandleAction(ctx core.ActionContext) error                 { return nil }
func (c *ApplyRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return 200, nil }
func (c *ApplyRun) Cancel(ctx core.ExecutionContext) error                    { return nil }
func (c *ApplyRun) Cleanup(ctx core.SetupContext) error                       { return nil }
func (c *ApplyRun) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "default", Label: "Run Applied", Description: "Emits when the run is successfully applied"},
	}
}
func (c *ApplyRun) DefaultOutputChannel() core.OutputChannel { return core.DefaultOutputChannel }
func (c *ApplyRun) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":  "run-xxxxxx",
		"action": "applied",
	}
}
func (c *ApplyRun) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *ApplyRun) Documentation() string { return "" }
