package azure

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type StopVMComponent struct {
}

type StopVMConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	Name          string `json:"name" mapstructure:"name"`
}

func (c *StopVMComponent) Name() string {
	return "azure.stopVirtualMachine"
}

func (c *StopVMComponent) Label() string {
	return "Stop Virtual Machine"
}

func (c *StopVMComponent) Description() string {
	return "Powers off an Azure Virtual Machine without deallocating"
}

func (c *StopVMComponent) Documentation() string {
	return `
The Stop Virtual Machine component powers off an existing Azure VM without deallocating it.

## Use Cases

- **Temporary shutdown**: Stop a VM temporarily while keeping its compute allocation
- **Maintenance windows**: Power off VMs during maintenance periods
- **Pre-deallocate workflows**: Stop VMs before performing other operations

## How It Works

1. Validates the VM parameters
2. Initiates VM power-off via the Azure Compute API
3. Waits for the VM to be fully powered off (using Azure's Long-Running Operation pattern)
4. Returns the VM details including ID and name

## Configuration

- **Resource Group**: The Azure resource group containing the VM
- **VM Name**: The name of the virtual machine to stop

## Output

Returns the stopped VM information including:
- **id**: The Azure resource ID of the VM
- **name**: The name of the VM
- **resourceGroup**: The resource group containing the VM

## Notes

- The VM is powered off but compute resources remain allocated — **you still incur compute charges**
- To fully release compute resources and stop charges, use the "Deallocate Virtual Machine" action
- The operation is a Long-Running Operation (LRO) that typically takes 1-2 minutes
- The component waits for the VM to be fully powered off before completing
`
}

func (c *StopVMComponent) Icon() string {
	return "azure"
}

func (c *StopVMComponent) Color() string {
	return "blue"
}

func (c *StopVMComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":            "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
		"name":          "my-vm",
		"resourceGroup": "my-rg",
	}
}

func (c *StopVMComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *StopVMComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceGroup",
			Label:       "Resource Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Azure resource group containing the VM",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeResourceGroupDropdown,
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "name",
			Label:       "VM Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name of the virtual machine to stop",
			Placeholder: "my-vm",
		},
	}
}

func (c *StopVMComponent) Setup(ctx core.SetupContext) error {
	config := StopVMConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
	}
	if config.Name == "" {
		return fmt.Errorf("VM name is required")
	}

	return nil
}

func (c *StopVMComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *StopVMComponent) Execute(ctx core.ExecutionContext) error {
	config := StopVMConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	req := VMActionRequest{
		ResourceGroup: config.ResourceGroup,
		VMName:        config.Name,
	}

	ctx.Logger.Infof("Stopping Azure VM: %s in resource group %s", config.Name, config.ResourceGroup)
	output, err := StopVM(context.Background(), provider, req, ctx.Logger)
	if err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"azure.vm",
		[]any{output},
	)
}

func (c *StopVMComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *StopVMComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *StopVMComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *StopVMComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *StopVMComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
