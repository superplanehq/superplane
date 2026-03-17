package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const snapshotPollInterval = 5 * time.Second

type CreateSnapshot struct{}

type CreateSnapshotSpec struct {
	Droplet string `json:"droplet" mapstructure:"droplet"`
	Name    string `json:"name" mapstructure:"name"`
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
	return `The Create Snapshot component creates a point-in-time snapshot of a DigitalOcean Droplet.

## Use Cases

- **Backup**: Create a backup before performing risky operations on a droplet
- **Image creation**: Create a custom image from an existing droplet for reuse
- **Migration**: Snapshot a droplet before migrating to a different region or size

## Configuration

- **Droplet**: The ID of the droplet to snapshot (required)
- **Name**: A human-readable name for the snapshot (required)

## Output

Returns the snapshot details including:
- **id**: Snapshot ID
- **name**: Snapshot name
- **created_at**: When the snapshot was created
- **resource_id**: The ID of the droplet that was snapshotted
- **regions**: Regions where the snapshot is available
- **min_disk_size**: Minimum disk size required to use this snapshot
- **size_gigabytes**: Size of the snapshot in GB`
}

func (c *CreateSnapshot) Icon() string {
	return "camera"
}

func (c *CreateSnapshot) Color() string {
	return "gray"
}

func (c *CreateSnapshot) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateSnapshot) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "droplet",
			Label:       "Droplet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The droplet to snapshot",
			Placeholder: "Select droplet",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "droplet",
					UseNameAsValue: false,
				},
			},
		},
		{
			Name:        "name",
			Label:       "Snapshot Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A human-readable name for the snapshot",
		},
	}
}

func (c *CreateSnapshot) Setup(ctx core.SetupContext) error {
	spec := CreateSnapshotSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Droplet == "" {
		return errors.New("droplet is required")
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	err = resolveDropletMetadata(ctx, spec.Droplet)
	if err != nil {
		return fmt.Errorf("error resolving droplet metadata: %v", err)
	}

	return nil
}

func (c *CreateSnapshot) Execute(ctx core.ExecutionContext) error {
	spec := CreateSnapshotSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	dropletID, err := strconv.Atoi(spec.Droplet)
	if err != nil {
		return fmt.Errorf("invalid droplet ID %q: must be a number", spec.Droplet)
	}
	action, err := client.CreateDropletSnapshot(dropletID, spec.Name)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %v", err)
	}

	err = ctx.Metadata.Set(map[string]any{
		"actionID": action.ID,
		"droplet":  dropletID,
	})
	if err != nil {
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
		ActionID int `mapstructure:"actionID"`
		Droplet  int `mapstructure:"droplet"`
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
		return fmt.Errorf("failed to get action: %v", err)
	}

	switch action.Status {
	case "completed":
		// Fetch the most recent snapshot for this droplet to get full details
		snapshots, err := client.GetDropletSnapshots(metadata.Droplet)
		if err != nil {
			return fmt.Errorf("failed to get droplet snapshots: %v", err)
		}

		if len(snapshots) == 0 {
			return fmt.Errorf("no snapshots found for droplet %d after action completed", metadata.Droplet)
		}

		snapshot, err := latestSnapshot(snapshots)
		if err != nil {
			return fmt.Errorf("failed to determine latest snapshot: %v", err)
		}

		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.snapshot.created",
			[]any{snapshot},
		)
	case "in-progress":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, snapshotPollInterval)
	case "errored":
		return fmt.Errorf("snapshot action failed")
	default:
		return fmt.Errorf("snapshot action reached unexpected status %q", action.Status)
	}
}

func latestSnapshot(snapshots []Snapshot) (Snapshot, error) {
	latest := snapshots[0]
	latestTime, err := time.Parse(time.RFC3339, latest.CreatedAt)
	if err != nil {
		return Snapshot{}, fmt.Errorf("failed to parse created_at for snapshot %s: %v", latest.ID, err)
	}

	for _, s := range snapshots[1:] {
		t, err := time.Parse(time.RFC3339, s.CreatedAt)
		if err != nil {
			return Snapshot{}, fmt.Errorf("failed to parse created_at for snapshot %s: %v", s.ID, err)
		}

		if t.After(latestTime) {
			latest = s
			latestTime = t
		}
	}

	return latest, nil
}

func (c *CreateSnapshot) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateSnapshot) Cleanup(ctx core.SetupContext) error {
	return nil
}
