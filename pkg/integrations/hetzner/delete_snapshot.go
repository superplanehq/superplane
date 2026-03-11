package hetzner

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeleteSnapshotPayloadType = "hetzner.snapshot.deleted"

type DeleteSnapshot struct{}

type DeleteSnapshotSpec struct {
	Snapshot string `json:"snapshot" mapstructure:"snapshot"`
}

type DeleteSnapshotExecutionMetadata struct {
	ImageID string `json:"imageId" mapstructure:"imageId"`
}

func (c *DeleteSnapshot) Name() string {
	return "hetzner.deleteSnapshot"
}

func (c *DeleteSnapshot) Label() string {
	return "Delete Snapshot"
}

func (c *DeleteSnapshot) Description() string {
	return "Delete a Hetzner Cloud snapshot image"
}

func (c *DeleteSnapshot) Documentation() string {
	return `The Delete Snapshot component deletes a snapshot image in Hetzner Cloud.

## How It Works

1. Deletes the selected snapshot via the Hetzner API
2. Emits on the default output when the snapshot is deleted. If deletion fails, the execution errors.
`
}

func (c *DeleteSnapshot) Icon() string {
	return "hetzner"
}

func (c *DeleteSnapshot) Color() string {
	return "gray"
}

func (c *DeleteSnapshot) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteSnapshot) ExampleOutput() map[string]any {
	return map[string]any{
		"imageId": "67890",
	}
}

func (c *DeleteSnapshot) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "snapshot",
			Label:    "Snapshot",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "snapshot_image",
				},
			},
			Description: "Snapshot image to delete",
		},
	}
}

func (c *DeleteSnapshot) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteSnapshot) Setup(ctx core.SetupContext) error {
	spec := DeleteSnapshotSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.Snapshot) == "" {
		return fmt.Errorf("snapshot is required")
	}
	return nil
}

func (c *DeleteSnapshot) Execute(ctx core.ExecutionContext) error {
	imageID, err := resolveImageID(ctx.Configuration, "snapshot")
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(DeleteSnapshotExecutionMetadata{ImageID: imageID}); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.DeleteImage(imageID); err != nil {
		return fmt.Errorf("delete snapshot: %w", err)
	}

	payload := map[string]any{
		"imageId": imageID,
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteSnapshotPayloadType, []any{payload})
}

func (c *DeleteSnapshot) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteSnapshot) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteSnapshot) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *DeleteSnapshot) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteSnapshot) Cleanup(ctx core.SetupContext) error {
	return nil
}
