package flyio

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeleteMachinePayloadType = "flyio.machine"

type DeleteMachine struct{}

type DeleteMachineSpec struct {
	App     string `json:"app" mapstructure:"app"`
	Machine string `json:"machine" mapstructure:"machine"`
	Force   bool   `json:"force" mapstructure:"force"`
}

func (c *DeleteMachine) Name() string {
	return "flyio.deleteMachine"
}

func (c *DeleteMachine) Label() string {
	return "Delete Machine"
}

func (c *DeleteMachine) Description() string {
	return "Permanently delete a Fly.io Machine"
}

func (c *DeleteMachine) Documentation() string {
	return `The Delete Machine component permanently deletes a Fly.io Machine.

## Use Cases

- **Ephemeral task cleanup**: Delete a machine after it finishes a one-off job
- **Decommission**: Remove machines that are no longer needed as part of a workflow
- **Force removal**: Force-delete a machine that is stuck and cannot be stopped normally

## How It Works

1. Calls the Fly.io Machines API to delete the selected Machine
2. Emits a payload confirming the deletion

## Configuration

- **App**: The Fly.io application that owns the Machine
- **Machine**: The specific Machine to delete (filtered by the selected App)
- **Force** (optional): Force-delete the machine even if it is currently running

## Notes

- Deletion is immediate and irreversible
- Enable **Force** to delete a running machine without stopping it first`
}

func (c *DeleteMachine) Icon() string {
	return "trash-2"
}

func (c *DeleteMachine) Color() string {
	return "red"
}

func (c *DeleteMachine) ExampleOutput() map[string]any {
	return map[string]any{
		"machineId": "148ed726c17589",
		"appName":   "my-fly-app",
		"deleted":   true,
	}
}

func (c *DeleteMachine) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteMachine) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "app",
			Label:    "App",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "app",
					UseNameAsValue: true,
				},
			},
			Description: "Fly.io application that owns the Machine",
		},
		{
			Name:     "machine",
			Label:    "Machine",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "machine",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "app",
							ValueFrom: &configuration.ParameterValueFrom{Field: "app"},
						},
					},
				},
			},
			Description: "Machine to delete",
		},
		{
			Name:        "force",
			Label:       "Force Delete",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Force-delete the machine even if it is currently running",
		},
	}
}

func decodeDeleteMachineSpec(configuration any) (DeleteMachineSpec, error) {
	spec := DeleteMachineSpec{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return DeleteMachineSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.App = strings.TrimSpace(spec.App)
	if spec.App == "" {
		return DeleteMachineSpec{}, fmt.Errorf("app is required")
	}

	spec.Machine = strings.TrimSpace(spec.Machine)
	if spec.Machine == "" {
		return DeleteMachineSpec{}, fmt.Errorf("machine is required")
	}

	return spec, nil
}


func (c *DeleteMachine) Setup(ctx core.SetupContext) error {
	_, err := decodeDeleteMachineSpec(ctx.Configuration)
	return err
}

func (c *DeleteMachine) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteMachine) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeDeleteMachineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	machineID := parseMachineID(spec.Machine)
	if err := client.DeleteMachine(spec.App, machineID, spec.Force); err != nil {
		return fmt.Errorf("failed to delete machine: %w", err)
	}

	ctx.Logger.Infof("Machine %s/%s deleted", spec.App, machineID)

	payload := map[string]any{
		"machineId": machineID,
		"appName":   spec.App,
		"deleted":   true,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteMachinePayloadType, []any{payload})
}

func (c *DeleteMachine) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *DeleteMachine) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *DeleteMachine) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteMachine) HandleAction(_ core.ActionContext) error {
	return nil
}

func (c *DeleteMachine) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
