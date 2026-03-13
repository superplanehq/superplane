package digitalocean

import (
	"errors"
	"fmt"
	"net/http"

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
	return `The Delete Snapshot component deletes a DigitalOcean snapshot image.

## Use Cases

- **Cleanup**: Remove old snapshots to free up storage and reduce costs
- **Lifecycle management**: Automatically delete snapshots after they are no longer needed
- **Rotation**: Delete older snapshots as part of a snapshot rotation policy

## Configuration

- **Snapshot ID**: The ID of the snapshot to delete (required)

## Output

Returns confirmation of the deleted snapshot including:
- **snapshotId**: The ID of the deleted snapshot
- **deleted**: Confirmation that the snapshot was deleted`
}

func (c *DeleteSnapshot) Icon() string {
	return "trash"
}

func (c *DeleteSnapshot) Color() string {
	return "gray"
}

func (c *DeleteSnapshot) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteSnapshot) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "snapshotId",
			Label:       "Snapshot ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the snapshot to delete",
		},
	}
}

func (c *DeleteSnapshot) Setup(ctx core.SetupContext) error {
	spec := DeleteSnapshotSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.SnapshotID == "" {
		return errors.New("snapshot ID is required")
	}

	return nil
}

func (c *DeleteSnapshot) Execute(ctx core.ExecutionContext) error {
	spec := DeleteSnapshotSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.DeleteSnapshot(spec.SnapshotID)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %v", err)
	}

	result := map[string]any{
		"snapshotId": spec.SnapshotID,
		"deleted":    true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.snapshot.deleted",
		[]any{result},
	)
}

func (c *DeleteSnapshot) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteSnapshot) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteSnapshot) Actions() []core.Action {
	return nil
}

func (c *DeleteSnapshot) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteSnapshot) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteSnapshot) Cleanup(ctx core.SetupContext) error {
	return nil
}
