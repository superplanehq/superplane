package daytona

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeleteSandboxPayloadType = "daytona.delete.response"

type DeleteSandbox struct{}

type DeleteSandboxSpec struct {
	Sandbox string `json:"sandbox"`
	Force   bool   `json:"force,omitempty"`
}

type DeleteSandboxPayload struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}

func (d *DeleteSandbox) Name() string {
	return "daytona.deleteSandbox"
}

func (d *DeleteSandbox) Label() string {
	return "Delete Sandbox"
}

func (d *DeleteSandbox) Description() string {
	return "Delete a sandbox environment"
}

func (d *DeleteSandbox) Documentation() string {
	return `The Delete Sandbox component removes an existing Daytona sandbox.

## Use Cases

- **Resource cleanup**: Delete sandboxes after code execution is complete
- **Cost management**: Remove unused sandboxes to free resources
- **Workflow cleanup**: Clean up sandboxes at the end of automation workflows

## Configuration

- **Sandbox**: The ID or name of the sandbox to delete (from createSandbox output)
- **Force**: Optional flag to force deletion even if sandbox is running

## Output

Returns deletion confirmation including:
- **deleted**: Boolean indicating successful deletion
- **id**: The ID of the deleted sandbox

## Notes

- Always delete sandboxes when they are no longer needed
- Sandboxes will auto-stop after the configured interval, but explicit deletion frees resources immediately`
}

func (d *DeleteSandbox) Icon() string {
	return "daytona"
}

func (d *DeleteSandbox) Color() string {
	return "orange"
}

func (d *DeleteSandbox) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteSandbox) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "sandbox",
			Label:       "Sandbox",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The ID or name of the sandbox to delete",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "sandbox",
				},
			},
		},
		{
			Name:        "force",
			Label:       "Force Delete",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Force deletion even if sandbox is running",
		},
	}
}

func (d *DeleteSandbox) Setup(ctx core.SetupContext) error {
	spec := DeleteSandboxSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Sandbox == "" {
		return fmt.Errorf("sandbox is required")
	}

	return nil
}

func (d *DeleteSandbox) Execute(ctx core.ExecutionContext) error {
	spec := DeleteSandboxSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	err = client.DeleteSandbox(spec.Sandbox, spec.Force)
	if err != nil {
		return fmt.Errorf("failed to delete sandbox: %v", err)
	}

	payload := DeleteSandboxPayload{
		Deleted: true,
		ID:      spec.Sandbox,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeleteSandboxPayloadType,
		[]any{payload},
	)
}

func (d *DeleteSandbox) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteSandbox) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteSandbox) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteSandbox) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteSandbox) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (d *DeleteSandbox) Cleanup(ctx core.SetupContext) error {
	return nil
}
