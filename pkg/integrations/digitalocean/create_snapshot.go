package digitalocean

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const snapshotPollInterval = 10 * time.Second

type CreateSnapshot struct{}

type CreateSnapshotSpec struct {
	DropletID string `json:"dropletId" mapstructure:"dropletId"`
	Name      string `json:"name" mapstructure:"name"`
}

func (c *CreateSnapshot) Name() string {
	return "digitalocean.createSnapshot"
}

func (c *CreateSnapshot) Label() string {
	return "Create Snapshot"
}

func (c *CreateSnapshot) Description() string {
	return "Create a snapshot of a DigitalOcean Droplet"
}

func (c *CreateSnapshot) Documentation() string {
	return `The Create Snapshot component creates a snapshot image from a DigitalOcean Droplet and waits for completion.

## How It Works

1. Sends a snapshot action to the specified droplet
2. Polls the action until snapshot creation finishes
3. Emits the action details on the default output. If creation fails, the execution errors.

## Configuration

- **Droplet ID**: The ID of the droplet to snapshot (required, supports expressions)
- **Snapshot Name**: A name for the snapshot (optional, supports expressions)

## Output

Returns the snapshot action details:
- **actionId**: The DigitalOcean action ID
- **dropletId**: The source droplet ID
- **status**: The final status of the action`
}

func (c *CreateSnapshot) Icon() string {
	return "server"
}

func (c *CreateSnapshot) Color() string {
	return "gray"
}

func (c *CreateSnapshot) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateSnapshot) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"actionId":  12345,
			"dropletId": 98765432,
			"status":    "completed",
		},
	}
}

func (c *CreateSnapshot) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dropletId",
			Label:       "Droplet ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The ID of the droplet to snapshot",
		},
		{
			Name:        "name",
			Label:       "Snapshot Name",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Togglable:   true,
			Description: "A name for the snapshot",
		},
	}
}

func (c *CreateSnapshot) Setup(ctx core.SetupContext) error {
	spec := CreateSnapshotSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.DropletID == "" {
		return fmt.Errorf("dropletId is required")
	}

	return nil
}

func (c *CreateSnapshot) Execute(ctx core.ExecutionContext) error {
	spec := CreateSnapshotSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	dropletID, err := resolveIntID(ctx.Configuration, "dropletId")
	if err != nil {
		return err
	}

	snapshotName := readStringFromAny(spec.Name)

	if err := ctx.Metadata.Set(map[string]any{"dropletId": dropletID}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	action, err := client.CreateDropletSnapshot(dropletID, snapshotName)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %v", err)
	}

	if err := ctx.Metadata.Set(map[string]any{
		"dropletId": dropletID,
		"actionId":  action.ID,
	}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, snapshotPollInterval)
}

func (c *CreateSnapshot) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateSnapshot) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateSnapshot) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *CreateSnapshot) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata struct {
		DropletID int `mapstructure:"dropletId"`
		ActionID  int `mapstructure:"actionId"`
	}

	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	action, err := client.GetAction(metadata.ActionID)
	if err != nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, snapshotPollInterval)
	}

	switch action.Status {
	case "completed":
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.snapshot.created",
			[]any{map[string]any{
				"actionId":  metadata.ActionID,
				"dropletId": metadata.DropletID,
				"status":    "completed",
			}},
		)
	case "in-progress":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, snapshotPollInterval)
	case "errored":
		return fmt.Errorf("snapshot creation failed for droplet %d", metadata.DropletID)
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, snapshotPollInterval)
	}
}

func (c *CreateSnapshot) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateSnapshot) Cleanup(ctx core.SetupContext) error {
	return nil
}
