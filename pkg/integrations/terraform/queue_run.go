package terraform

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueueRun struct{}
type QueueRunSpec struct {
	WorkspaceID string `json:"workspaceId"`
	Message     string `json:"message"`
}

func (c *QueueRun) Name() string        { return "terraform.queueRun" }
func (c *QueueRun) Label() string       { return "Queue Run" }
func (c *QueueRun) Description() string { return "Queues a new run for a specific Workspace." }
func (c *QueueRun) Icon() string        { return "play-circle" }
func (c *QueueRun) Color() string       { return "purple" }
func (c *QueueRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "workspaceId",
			Label:    "Workspace ID",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "message",
			Label:    "Run Message",
			Type:     configuration.FieldTypeString,
			Required: false,
		},
	}
}
func (c *QueueRun) Setup(ctx core.SetupContext) error { return nil }
func (c *QueueRun) Execute(ctx core.ExecutionContext) error {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	spec := QueueRunSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	resolvedWsID, err := client.ResolveWorkspaceID(context.Background(), spec.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	msg := fmt.Sprintf("⚙ %s", spec.Message)

	run, err := client.CreateRun(context.Background(), resolvedWsID, msg)
	if err != nil {
		return fmt.Errorf("failed to queue run: %w", err)
	}

	return ctx.ExecutionState.Emit("default", "", []any{
		map[string]any{
			"runId":  run.ID,
			"status": run.Attributes.Status,
		},
	})
}

func (c *QueueRun) Actions() []core.Action                                    { return nil }
func (c *QueueRun) HandleAction(ctx core.ActionContext) error                 { return nil }
func (c *QueueRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return 200, nil }
func (c *QueueRun) Cancel(ctx core.ExecutionContext) error                    { return nil }
func (c *QueueRun) Cleanup(ctx core.SetupContext) error                       { return nil }
func (c *QueueRun) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "default", Label: "Run Queued", Description: "Emits when a new run is successfully queued"},
	}
}
func (c *QueueRun) DefaultOutputChannel() core.OutputChannel { return core.DefaultOutputChannel }
func (c *QueueRun) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":  "run-xxxxxx",
		"status": "pending",
	}
}
func (c *QueueRun) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *QueueRun) Documentation() string { return "" }
