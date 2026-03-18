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

type DeallocateVMComponent struct {
	integration *AzureIntegration
}

type DeallocateVMConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	Name          string `json:"name" mapstructure:"name"`
}

func (c *DeallocateVMComponent) Name() string {
	return "azure.deallocateVirtualMachine"
}

func (c *DeallocateVMComponent) Label() string {
	return "Deallocate Virtual Machine"
}

func (c *DeallocateVMComponent) Description() string {
	return "Deallocates an Azure Virtual Machine, releasing compute resources"
}

func (c *DeallocateVMComponent) Documentation() string {
	return `
The Deallocate Virtual Machine component deallocates an existing Azure VM, stopping it and
releasing its compute resources.

## Use Cases

- **Cost optimization**: Deallocate VMs to stop compute charges during off-hours
- **Scheduled shutdown**: Deallocate VMs on a schedule to reduce costs
- **Pre-resize operations**: Deallocate a VM before changing its size
- **Environment teardown**: Deallocate VMs without deleting them

## How It Works

1. Validates the VM parameters
2. Initiates VM deallocation via the Azure Compute API
3. Waits for the VM to be fully deallocated (using Azure's Long-Running Operation pattern)
4. Returns the VM details including ID and name

## Configuration

- **Resource Group**: The Azure resource group containing the VM
- **VM Name**: The name of the virtual machine to deallocate

## Output

Returns the deallocated VM information including:
- **id**: The Azure resource ID of the VM
- **name**: The name of the VM
- **resourceGroup**: The resource group containing the VM

## Notes

- Deallocation stops the VM and releases compute resources — **compute charges stop**
- Only storage charges remain for the VM's disks
- Dynamic public IP addresses are released on deallocation
- The operation is a Long-Running Operation (LRO) that typically takes 1-3 minutes
- The component waits for the VM to be fully deallocated before completing
- To power off without releasing compute resources, use the "Stop Virtual Machine" action
`
}

func (c *DeallocateVMComponent) Icon() string {
	return "azure"
}

func (c *DeallocateVMComponent) Color() string {
	return "blue"
}

func (c *DeallocateVMComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":            "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
		"name":          "my-vm",
		"resourceGroup": "my-rg",
	}
}

func (c *DeallocateVMComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeallocateVMComponent) Configuration() []configuration.Field {
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
			Description: "The name of the virtual machine to deallocate",
			Placeholder: "my-vm",
		},
	}
}

func (c *DeallocateVMComponent) Setup(ctx core.SetupContext) error {
	config := DeallocateVMConfiguration{}
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

func (c *DeallocateVMComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeallocateVMComponent) Execute(ctx core.ExecutionContext) error {
	config := DeallocateVMConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := c.integration.ensureProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	req := VMActionRequest{
		ResourceGroup: config.ResourceGroup,
		VMName:        config.Name,
	}

	ctx.Logger.Infof("Deallocating Azure VM: %s in resource group %s", config.Name, config.ResourceGroup)
	output, err := DeallocateVM(context.Background(), provider, req, ctx.Logger)
	if err != nil {
		return fmt.Errorf("failed to deallocate VM: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"azure.vm",
		[]any{output},
	)
}

func (c *DeallocateVMComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeallocateVMComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeallocateVMComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *DeallocateVMComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeallocateVMComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
