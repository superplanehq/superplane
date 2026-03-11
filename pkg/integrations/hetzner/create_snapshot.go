package hetzner

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	CreateSnapshotPayloadType  = "hetzner.snapshot.created"
	CreateSnapshotPollInterval = 5 * time.Second
)

type CreateSnapshot struct{}

type CreateSnapshotSpec struct {
	Server      string `json:"server" mapstructure:"server"`
	Description string `json:"description" mapstructure:"description"`
}

type CreateSnapshotExecutionMetadata struct {
	ActionID string         `json:"actionId" mapstructure:"actionId"`
	ServerID string         `json:"serverId" mapstructure:"serverId"`
	ImageID  string         `json:"imageId" mapstructure:"imageId"`
	Image    *ImageResponse `json:"image,omitempty" mapstructure:"image"`
}

func (c *CreateSnapshot) Name() string {
	return "hetzner.createSnapshot"
}

func (c *CreateSnapshot) Label() string {
	return "Create Snapshot"
}

func (c *CreateSnapshot) Description() string {
	return "Create a server snapshot in Hetzner Cloud and wait for completion"
}

func (c *CreateSnapshot) Documentation() string {
	return `The Create Snapshot component creates a snapshot image from an existing Hetzner Cloud server and waits for completion.

## How It Works

1. Calls the Hetzner API to create a snapshot from the selected server
2. Polls the action until snapshot creation finishes
3. Emits snapshot details (including image ID) on success. If creation fails, the execution errors.

## Configuration

- **Server**: Existing server to snapshot
- **Snapshot name** (optional): Snapshot description/name in Hetzner Cloud
`
}

func (c *CreateSnapshot) Icon() string {
	return "hetzner"
}

func (c *CreateSnapshot) Color() string {
	return "gray"
}

func (c *CreateSnapshot) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateSnapshot) ExampleOutput() map[string]any {
	return map[string]any{
		"actionId":     "12345",
		"imageId":      67890,
		"imageType":    "snapshot",
		"serverId":     "42",
		"snapshotName": "workflow-snapshot",
	}
}

func (c *CreateSnapshot) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "server",
			Label:    "Server",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "server",
				},
			},
			Description: "Server to snapshot",
		},
		{
			Name:        "description",
			Label:       "Snapshot name",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Snapshot name/description in Hetzner Cloud",
		},
	}
}

func (c *CreateSnapshot) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateSnapshot) Setup(ctx core.SetupContext) error {
	spec := CreateSnapshotSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.Server) == "" {
		return fmt.Errorf("server is required")
	}
	return nil
}

func (c *CreateSnapshot) Execute(ctx core.ExecutionContext) error {
	spec := CreateSnapshotSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	serverID, err := resolveServerID(ctx.Configuration)
	if err != nil {
		return err
	}
	description := strings.TrimSpace(readStringFromAny(spec.Description))

	if err := ctx.Metadata.Set(CreateSnapshotExecutionMetadata{ServerID: serverID}); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	image, action, err := client.CreateServerSnapshot(serverID, description)
	if err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	imageID := ""
	if image != nil {
		imageID = fmt.Sprintf("%d", image.ID)
	}

	if err := ctx.Metadata.Set(CreateSnapshotExecutionMetadata{
		ActionID: action.ID,
		ServerID: serverID,
		ImageID:  imageID,
		Image:    image,
	}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateSnapshotPollInterval)
}

func (c *CreateSnapshot) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (c *CreateSnapshot) HandleAction(ctx core.ActionContext) error {
	if ctx.Name == "poll" {
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *CreateSnapshot) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata CreateSnapshotExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("decode metadata: %w", err)
	}
	if metadata.ActionID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	action, err := client.GetAction(metadata.ActionID)
	if err != nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateSnapshotPollInterval)
	}

	switch action.Status {
	case ActionStatusRunning:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateSnapshotPollInterval)
	case ActionStatusError:
		msg := "create snapshot action failed"
		if action.Error != nil && action.Error.Message != "" {
			msg = action.Error.Message
		}
		return fmt.Errorf("%s", msg)
	case ActionStatusSuccess:
		image := metadata.Image
		if metadata.ImageID != "" {
			if refreshed, getErr := client.GetImage(metadata.ImageID); getErr == nil {
				image = refreshed
			}
		}
		payload := snapshotToPayload(metadata.ActionID, metadata.ServerID, image)
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateSnapshotPayloadType, []any{payload})
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateSnapshotPollInterval)
	}
}

func (c *CreateSnapshot) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreateSnapshot) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateSnapshot) Cleanup(ctx core.SetupContext) error {
	return nil
}

func snapshotToPayload(actionID, serverID string, image *ImageResponse) map[string]any {
	payload := map[string]any{
		"actionId": actionID,
		"serverId": serverID,
	}
	if image == nil {
		return payload
	}

	payload["imageId"] = image.ID
	payload["imageType"] = image.Type
	if strings.TrimSpace(image.Description) != "" {
		payload["snapshotName"] = image.Description
	}
	if strings.TrimSpace(image.Name) != "" {
		payload["imageName"] = image.Name
	}
	return payload
}
