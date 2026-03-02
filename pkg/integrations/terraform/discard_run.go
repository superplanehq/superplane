package terraform

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DiscardRun struct{}
type DiscardRunSpec struct {
	RunID   string `json:"runId"`
	Comment string `json:"comment"`
}

func (c *DiscardRun) Name() string        { return "terraform.discardRun" }
func (c *DiscardRun) Label() string       { return "Discard Run" }
func (c *DiscardRun) Color() string       { return "red" }
func (c *DiscardRun) Icon() string        { return "x-circle" }
func (c *DiscardRun) Description() string { return "Discards a pending or planned run." }
func (c *DiscardRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "runId", Label: "Run ID", Type: configuration.FieldTypeString, Required: true},
		{Name: "comment", Label: "Comment", Type: configuration.FieldTypeString, Required: false},
	}
}
func (c *DiscardRun) Execute(ctx core.ExecutionContext) error {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	spec := DiscardRunSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	err = client.DiscardRun(context.Background(), spec.RunID, spec.Comment)
	if err != nil {
		return fmt.Errorf("failed to discard run: %w", err)
	}
	return ctx.ExecutionState.Emit("default", "", []any{map[string]any{"runId": spec.RunID, "action": "discarded"}})
}

func (c *DiscardRun) Actions() []core.Action                                    { return nil }
func (c *DiscardRun) HandleAction(ctx core.ActionContext) error                 { return nil }
func (c *DiscardRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return 200, nil }
func (c *DiscardRun) Cancel(ctx core.ExecutionContext) error                    { return nil }
func (c *DiscardRun) Setup(ctx core.SetupContext) error                         { return nil }
func (c *DiscardRun) Cleanup(ctx core.SetupContext) error                       { return nil }
func (c *DiscardRun) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "default", Label: "Run Discarded", Description: "Emits when the run is successfully discarded"},
	}
}
func (c *DiscardRun) DefaultOutputChannel() core.OutputChannel { return core.DefaultOutputChannel }
func (c *DiscardRun) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":  "run-xxxxxx",
		"action": "discarded",
	}
}
func (c *DiscardRun) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *DiscardRun) Documentation() string { return "" }
