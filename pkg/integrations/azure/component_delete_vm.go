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

type DeleteVMComponent struct {
	integration *AzureIntegration
}

type DeleteVMConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	Name          string `json:"name" mapstructure:"name"`
}

func (c *DeleteVMComponent) Name() string {
	return "azure.deleteVirtualMachine"
}

func (c *DeleteVMComponent) Label() string {
	return "Delete Virtual Machine"
}

func (c *DeleteVMComponent) Description() string {
	return "Deletes an existing Azure Virtual Machine"
}

func (c *DeleteVMComponent) Documentation() string {
	return `
The Delete Virtual Machine component deletes an existing Azure VM.

## Use Cases

- **Infrastructure teardown**: Remove VMs as part of decommissioning workflows
- **Cost optimization**: Delete unused or idle VMs to reduce costs
- **Environment cleanup**: Remove temporary VMs after testing or development
- **Auto-scaling**: Delete VMs in response to reduced load

## How It Works

1. Validates the VM deletion parameters
2. Initiates VM deletion via the Azure Compute API
3. Waits for the VM to be fully deleted (using Azure's Long-Running Operation pattern)
4. Returns the deleted VM details including ID and name

## Configuration

- **Resource Group**: The Azure resource group containing the VM
- **VM Name**: The name of the virtual machine to delete

## Output

Returns the deleted VM information including:
- **id**: The Azure resource ID of the deleted VM
- **name**: The name of the deleted VM
- **resourceGroup**: The resource group that contained the VM

## Notes

- The VM deletion is a Long-Running Operation (LRO) that typically takes 1-3 minutes
- The component waits for the VM to be fully deleted before completing
- Deleting a VM does not delete associated resources (NICs, disks, public IPs)
- This operation is irreversible
`
}

func (c *DeleteVMComponent) Icon() string {
	return "azure"
}

func (c *DeleteVMComponent) Color() string {
	return "blue"
}

func (c *DeleteVMComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":            "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
		"name":          "my-vm",
		"resourceGroup": "my-rg",
	}
}

func (c *DeleteVMComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteVMComponent) Configuration() []configuration.Field {
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
			Description: "The name of the virtual machine to delete",
			Placeholder: "my-vm",
		},
	}
}

func (c *DeleteVMComponent) Setup(ctx core.SetupContext) error {
	config := DeleteVMConfiguration{}
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

func (c *DeleteVMComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteVMComponent) Execute(ctx core.ExecutionContext) error {
	config := DeleteVMConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := c.integration.ensureProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	req := DeleteVMRequest{
		ResourceGroup: config.ResourceGroup,
		VMName:        config.Name,
	}

	ctx.Logger.Infof("Deleting Azure VM: %s in resource group %s", config.Name, config.ResourceGroup)
	output, err := DeleteVM(context.Background(), provider, req, ctx.Logger)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"azure.vm",
		[]any{output},
	)
}

func (c *DeleteVMComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteVMComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteVMComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *DeleteVMComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteVMComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
