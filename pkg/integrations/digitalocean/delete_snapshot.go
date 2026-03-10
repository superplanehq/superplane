package digitalocean

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteSnapshot struct{}

type DeleteSnapshotSpec struct {
	SnapshotID string `json:"snapshotId" mapstructure:"snapshotId"`
}

func (c *DeleteSnapshot) Name() string {
	return "digitalocean.deleteSnapshot"
}

func (c *DeleteSnapshot) Label() string {
	return "Delete Snapshot"
}

func (c *DeleteSnapshot) Description() string {
	return "Delete a DigitalOcean snapshot"
}

func (c *DeleteSnapshot) Documentation() string {
	return `The Delete Snapshot component deletes a snapshot image in DigitalOcean.

## How It Works

1. Deletes the specified snapshot via the DigitalOcean API
2. Emits on the default output when the snapshot is deleted. If deletion fails, the execution errors.

## Configuration

- **Snapshot ID**: The ID of the snapshot to delete (required, supports expressions)

## Output

Returns confirmation of the deleted snapshot:
- **snapshotId**: The ID of the deleted snapshot`
}

func (c *DeleteSnapshot) Icon() string {
	return "server"
}

func (c *DeleteSnapshot) Color() string {
	return "gray"
}

func (c *DeleteSnapshot) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteSnapshot) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"snapshotId": "12345678",
		},
	}
}

func (c *DeleteSnapshot) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "snapshotId",
			Label:       "Snapshot ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The ID of the snapshot to delete",
		},
	}
}

func (c *DeleteSnapshot) Setup(ctx core.SetupContext) error {
	spec := DeleteSnapshotSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.SnapshotID == "" {
		return fmt.Errorf("snapshotId is required")
	}

	return nil
}

func (c *DeleteSnapshot) Execute(ctx core.ExecutionContext) error {
	snapshotID, err := resolveStringField(ctx.Configuration, "snapshotId")
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(map[string]any{"snapshotId": snapshotID}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.DeleteSnapshot(snapshotID); err != nil {
		return fmt.Errorf("failed to delete snapshot: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.snapshot.deleted",
		[]any{map[string]any{"snapshotId": snapshotID}},
	)
}

func (c *DeleteSnapshot) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteSnapshot) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteSnapshot) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteSnapshot) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteSnapshot) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *DeleteSnapshot) Cleanup(ctx core.SetupContext) error {
	return nil
}
