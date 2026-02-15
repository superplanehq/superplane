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
	DeleteServerPayloadType  = "hetzner.server.deleted"
	DeleteServerPollInterval = 5 * time.Second
)

type DeleteServer struct{}

type DeleteServerSpec struct {
	Server string `json:"server" mapstructure:"server"`
}

type DeleteServerExecutionMetadata struct {
	ActionID string `json:"actionId" mapstructure:"actionId"`
	ServerID string `json:"serverId" mapstructure:"serverId"`
}

func (c *DeleteServer) Name() string {
	return "hetzner.deleteServer"
}

func (c *DeleteServer) Label() string {
	return "Delete Server"
}

func (c *DeleteServer) Description() string {
	return "Delete a Hetzner Cloud server and wait for the action to complete"
}

func (c *DeleteServer) Documentation() string {
	return `The Delete Server component deletes a server in Hetzner Cloud and waits for the delete action to complete.

## How It Works

1. Deletes the selected server via the Hetzner API
2. Polls the API until the delete action finishes
3. Emits on the default output when the server is deleted. If deletion fails, the execution errors.
`
}

func (c *DeleteServer) Icon() string {
	return "hetzner"
}

func (c *DeleteServer) Color() string {
	return "gray"
}

func (c *DeleteServer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteServer) ExampleOutput() map[string]any {
	return map[string]any{
		"actionId": 123,
		"serverId": 42,
	}
}

func (c *DeleteServer) Configuration() []configuration.Field {
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
			Description: "Server to delete",
		},
	}
}

func (c *DeleteServer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteServer) Setup(ctx core.SetupContext) error {
	spec := DeleteServerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.Server) == "" {
		return fmt.Errorf("server is required")
	}
	return nil
}

func (c *DeleteServer) Execute(ctx core.ExecutionContext) error {
	spec := DeleteServerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}
	serverID, err := resolveServerID(ctx.Configuration)
	if err != nil {
		return err
	}

	// Store server ID in metadata early so it's visible in the UI
	// even if the API call fails.
	if err := ctx.Metadata.Set(DeleteServerExecutionMetadata{ServerID: serverID}); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	action, err := client.DeleteServer(serverID)
	if err != nil {
		return fmt.Errorf("delete server: %w", err)
	}

	if err := ctx.Metadata.Set(DeleteServerExecutionMetadata{
		ActionID: action.ID,
		ServerID: serverID,
	}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeleteServerPollInterval)
}

func (c *DeleteServer) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (c *DeleteServer) HandleAction(ctx core.ActionContext) error {
	if ctx.Name == "poll" {
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *DeleteServer) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata DeleteServerExecutionMetadata
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
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeleteServerPollInterval)
	}

	switch action.Status {
	case ActionStatusRunning:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeleteServerPollInterval)
	case ActionStatusError:
		msg := "delete server action failed"
		if action.Error != nil && action.Error.Message != "" {
			msg = action.Error.Message
		}
		return fmt.Errorf("%s", msg)
	case ActionStatusSuccess:
		payload := map[string]any{
			"actionId": metadata.ActionID,
			"serverId": metadata.ServerID,
		}
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteServerPayloadType, []any{payload})
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeleteServerPollInterval)
	}
}

func (c *DeleteServer) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *DeleteServer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteServer) Cleanup(ctx core.SetupContext) error {
	return nil
}
