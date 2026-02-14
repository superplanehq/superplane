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
	CreateServerPayloadType  = "hetzner.server.created"
	CreateServerPollInterval = 5 * time.Second
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
	ActionID string          `json:"actionId" mapstructure:"actionId"`
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
3. Emits the server details on the default output when ready. If creation fails, the execution errors.

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
	return "hetzner"
}

func (c *CreateServer) Color() string {
	return "gray"
}

func (c *CreateServer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateServer) ExampleOutput() map[string]any {
	return map[string]any{
		"id":       42,
		"name":     "my-server",
		"status":   "running",
		"created":  "2024-01-15T10:30:00+00:00",
		"publicIp": "1.2.3.4",
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
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select server type",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "server_type",
				},
			},
			Description: "Server type",
		},
		{
			Name:     "image",
			Label:    "Image",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "image",
				},
			},
			Description: "Image",
		},
		{
			Name:     "location",
			Label:    "Location",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "location",
					Parameters: []configuration.ParameterRef{
						{Name: "serverType", ValueFrom: &configuration.ParameterValueFrom{Field: "serverType"}},
					},
				},
			},
			Description: "Location (optional, omit for auto). Only locations that support the selected server type are shown.",
		},
		{
			Name:        "sshKeys",
			Label:       "SSH keys",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Add the name or ID of each SSH key from your Hetzner Cloud project (Security → SSH Keys). The server will allow login with these keys.",
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
	if metadata.ActionID == "" {
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
		msg := "create server action failed"
		if action.Error != nil && action.Error.Message != "" {
			msg = action.Error.Message
		}
		return fmt.Errorf("%s", msg)
	case ActionStatusSuccess:
		server := metadata.Server
		if server != nil && server.ID != "" {
			if refreshed, err := client.GetServer(server.ID); err == nil {
				server = refreshed
			}
		}
		payload := serverToPayload(server)
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateServerPayloadType, []any{payload})
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
		"id":      s.ID,
		"name":    s.Name,
		"status":  s.Status,
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
