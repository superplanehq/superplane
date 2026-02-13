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
	CreateServerPayloadType   = "hetzner.server.created"
	CreateServerSuccessChannel = "success"
	CreateServerFailedChannel  = "failed"
	CreateServerPollInterval   = 5 * time.Second
)

type CreateServer struct{}

type CreateServerSpec struct {
	Name       string   `json:"name" mapstructure:"name"`
	ServerType string   `json:"serverType" mapstructure:"serverType"`
	Image      string   `json:"image" mapstructure:"image"`
	Location   string   `json:"location" mapstructure:"location"`
	SSHKeys    []string `json:"sshKeys" mapstructure:"sshKeys"`
	UserData   string   `json:"userData" mapstructure:"userData"`
}

type CreateServerExecutionMetadata struct {
	ActionID int             `json:"actionId" mapstructure:"actionId"`
	Server   *ServerResponse `json:"server,omitempty" mapstructure:"server"`
}

func (c *CreateServer) Name() string {
	return "hetzner.createServer"
}

func (c *CreateServer) Label() string {
	return "Create Server"
}

func (c *CreateServer) Description() string {
	return "Create a Hetzner Cloud server and wait for the action to complete"
}

func (c *CreateServer) Documentation() string {
	return `The Create Server component creates a new server in Hetzner Cloud and waits for the create action to complete.

## How It Works

1. Creates a server with the given name, server type, image, and optional location/SSH keys/user data
2. Polls the Hetzner API until the create action finishes
3. **Success**: Emits the server details when the server is ready
4. **Failed**: Emits when the action fails

## Configuration

- **Name**: Server name (supports expressions)
- **Server type**: e.g. cx11, cpx11, cax11
- **Image**: Image name or ID, e.g. ubuntu-24.04
- **Location** (optional): e.g. fsn1, nbg1, hel1
- **SSH keys** (optional): List of SSH key names or IDs
- **User data** (optional): Cloud-init user data
`
}

func (c *CreateServer) Icon() string {
	return "server"
}

func (c *CreateServer) Color() string {
	return "gray"
}

func (c *CreateServer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: CreateServerSuccessChannel, Label: "Success"},
		{Name: CreateServerFailedChannel, Label: "Failed"},
	}
}

func (c *CreateServer) ExampleOutput() map[string]any {
	return map[string]any{
		"id":     42,
		"name":   "my-server",
		"status": "running",
		"public_net": map[string]any{
			"ipv4": map[string]any{"ip": "1.2.3.4"},
		},
	}
}

func (c *CreateServer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Server name",
		},
		{
			Name:        "serverType",
			Label:       "Server type",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Server type (e.g. cx11, cpx11)",
		},
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Image name or ID (e.g. ubuntu-24.04)",
		},
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Location (e.g. fsn1, nbg1). Omit for auto.",
		},
		{
			Name:     "sshKeys",
			Label:    "SSH keys",
			Type:     configuration.FieldTypeList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "SSH key",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "userData",
			Label:       "User data",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Cloud-init user data",
		},
	}
}

func (c *CreateServer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateServer) Setup(ctx core.SetupContext) error {
	spec := CreateServerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.ServerType) == "" {
		return fmt.Errorf("serverType is required")
	}
	if strings.TrimSpace(spec.Image) == "" {
		return fmt.Errorf("image is required")
	}
	return nil
}

func (c *CreateServer) Execute(ctx core.ExecutionContext) error {
	spec := CreateServerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}
	name := strings.TrimSpace(readStringFromAny(spec.Name))
	if name == "" {
		return fmt.Errorf("name is required")
	}
	serverType := strings.TrimSpace(spec.ServerType)
	image := strings.TrimSpace(spec.Image)
	location := strings.TrimSpace(spec.Location)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	server, action, err := client.CreateServer(name, serverType, image, location, spec.SSHKeys, spec.UserData)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	metadata := CreateServerExecutionMetadata{
		ActionID: action.ID,
		Server:   server,
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateServerPollInterval)
}

func (c *CreateServer) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (c *CreateServer) HandleAction(ctx core.ActionContext) error {
	if ctx.Name == "poll" {
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *CreateServer) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata CreateServerExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("decode metadata: %w", err)
	}
	if metadata.ActionID == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	action, err := client.GetAction(metadata.ActionID)
	if err != nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateServerPollInterval)
	}

	switch action.Status {
	case ActionStatusRunning:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateServerPollInterval)
	case ActionStatusError:
		msg := "action failed"
		if action.Error != nil && action.Error.Message != "" {
			msg = action.Error.Message
		}
		payload := map[string]any{
			"actionId": metadata.ActionID,
			"error":    msg,
			"server":   metadata.Server,
		}
		return ctx.ExecutionState.Emit(CreateServerFailedChannel, CreateServerPayloadType, []any{payload})
	case ActionStatusSuccess:
		payload := serverToPayload(metadata.Server)
		return ctx.ExecutionState.Emit(CreateServerSuccessChannel, CreateServerPayloadType, []any{payload})
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateServerPollInterval)
	}
}

func (c *CreateServer) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateServer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateServer) Cleanup(ctx core.SetupContext) error {
	return nil
}

func serverToPayload(s *ServerResponse) map[string]any {
	if s == nil {
		return map[string]any{}
	}
	out := map[string]any{
		"id":     s.ID,
		"name":   s.Name,
		"status": s.Status,
		"created": s.Created,
	}
	if s.PublicNet.IPv4.IP != "" {
		out["publicIp"] = s.PublicNet.IPv4.IP
	}
	return out
}

func readStringFromAny(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	case int:
		return fmt.Sprintf("%d", x)
	default:
		return fmt.Sprintf("%v", v)
	}
}
