package daytona

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const SandboxPayloadType = "daytona.sandbox"

type CreateSandbox struct{}

type CreateSandboxSpec struct {
	Snapshot         string `json:"snapshot,omitempty"`
	Target           string `json:"target,omitempty"`
	AutoStopInterval int    `json:"autoStopInterval,omitempty"`
}

type SandboxPayload struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

func (c *CreateSandbox) Name() string {
	return "daytona.createSandbox"
}

func (c *CreateSandbox) Label() string {
	return "Create Sandbox"
}

func (c *CreateSandbox) Description() string {
	return "Create an isolated sandbox environment for code execution"
}

func (c *CreateSandbox) Documentation() string {
	return `The Create Sandbox component creates an isolated environment for executing code safely.

## Use Cases

- **AI code execution**: Run AI-generated code in a secure sandbox
- **Code testing**: Test untrusted code without affecting your infrastructure
- **Development environments**: Create ephemeral development environments

## Configuration

- **Snapshot**: Base environment snapshot (optional, uses default if not specified)
- **Target**: Target region for the sandbox (optional)
- **Auto Stop Interval**: Time in minutes before the sandbox auto-stops

## Output

Returns the sandbox information including:
- **id**: The unique sandbox identifier (use this in subsequent execute/delete operations)
- **state**: The current state of the sandbox (e.g., "started")

## Notes

- Sandboxes are created in sub-90ms
- Each sandbox is fully isolated
- Remember to delete sandboxes when done to free resources`
}

func (c *CreateSandbox) Icon() string {
	return "box"
}

func (c *CreateSandbox) Color() string {
	return "orange"
}

func (c *CreateSandbox) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateSandbox) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "snapshot",
			Label:       "Snapshot",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Base environment snapshot for the sandbox (optional)",
			Placeholder: "default",
		},
		{
			Name:     "target",
			Label:    "Target Region",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "US", Value: "us"},
						{Label: "EU", Value: "eu"},
					},
				},
			},
			Description: "Target region for the sandbox",
		},
		{
			Name:        "autoStopInterval",
			Label:       "Auto Stop Interval",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Time in minutes before the sandbox auto-stops",
			Placeholder: "15",
		},
	}
}

func (c *CreateSandbox) Setup(ctx core.SetupContext) error {
	spec := CreateSandboxSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.AutoStopInterval < 0 {
		return fmt.Errorf("autoStopInterval must be a positive number")
	}

	return nil
}

func (c *CreateSandbox) Execute(ctx core.ExecutionContext) error {
	spec := CreateSandboxSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	req := &CreateSandboxRequest{
		Snapshot:         spec.Snapshot,
		Target:           spec.Target,
		AutoStopInterval: spec.AutoStopInterval,
	}

	sandbox, err := client.CreateSandbox(req)
	if err != nil {
		return fmt.Errorf("failed to create sandbox: %v", err)
	}

	payload := SandboxPayload{
		ID:    sandbox.ID,
		State: sandbox.State,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		SandboxPayloadType,
		[]any{payload},
	)
}

func (c *CreateSandbox) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateSandbox) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateSandbox) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateSandbox) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateSandbox) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
