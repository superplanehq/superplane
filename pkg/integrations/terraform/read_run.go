package terraform

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ReadRun struct{}
type ReadRunSpec struct {
	RunID string `json:"runId"`
}

func (c *ReadRun) Name() string  { return "terraform.readRun" }
func (c *ReadRun) Label() string { return "Read Run Details" }
func (c *ReadRun) Description() string {
	return "Retrieves comprehensive details and status about a run."
}
func (c *ReadRun) Icon() string  { return "info" }
func (c *ReadRun) Color() string { return "gray" }
func (c *ReadRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "runId", Label: "Run ID", Type: configuration.FieldTypeString, Required: true},
	}
}
func (c *ReadRun) Execute(ctx core.ExecutionContext) error {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	spec := ReadRunSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	run, err := client.ReadRun(context.Background(), spec.RunID)
	if err != nil {
		return fmt.Errorf("failed to read run: %w", err)
	}

	return ctx.ExecutionState.Emit("default", "", []any{
		map[string]any{
			"runId":     run.ID,
			"status":    run.Attributes.Status,
			"message":   run.Attributes.Message,
			"createdAt": run.Attributes.CreatedAt,
		},
	})
}

func (c *ReadRun) Actions() []core.Action                                    { return nil }
func (c *ReadRun) HandleAction(ctx core.ActionContext) error                 { return nil }
func (c *ReadRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return 200, nil }
func (c *ReadRun) Cancel(ctx core.ExecutionContext) error                    { return nil }
func (c *ReadRun) Cleanup(ctx core.SetupContext) error                       { return nil }
func (c *ReadRun) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "default", Label: "Run Details", Description: "Emits the run details"},
	}
}
func (c *ReadRun) DefaultOutputChannel() core.OutputChannel { return core.DefaultOutputChannel }
func (c *ReadRun) Setup(ctx core.SetupContext) error        { return nil }
func (c *ReadRun) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":     "run-xxxxxx",
		"status":    "planned",
		"message":   "Queued manually",
		"createdAt": "2024-01-01T12:00:00Z",
	}
}
func (c *ReadRun) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *ReadRun) Documentation() string { return "" }
