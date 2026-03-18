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

type RestartVMComponent struct {
}

type RestartVMConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	Name          string `json:"name" mapstructure:"name"`
}

func (c *RestartVMComponent) Name() string {
	return "azure.restartVirtualMachine"
}

func (c *RestartVMComponent) Label() string {
	return "Restart Virtual Machine"
}

func (c *RestartVMComponent) Description() string {
	return "Restarts an Azure Virtual Machine"
}

func (c *RestartVMComponent) Documentation() string {
	return `
The Restart Virtual Machine component restarts an existing Azure VM in place.

## Use Cases

- **Apply updates**: Restart VMs after applying OS or software updates
- **Troubleshooting**: Restart VMs to recover from transient issues
- **Configuration changes**: Restart VMs after configuration changes that require a reboot
- **Automated maintenance**: Restart VMs as part of maintenance workflows

## How It Works

1. Validates the VM parameters
2. Initiates VM restart via the Azure Compute API
3. Waits for the VM to be fully restarted (using Azure's Long-Running Operation pattern)
4. Returns the VM details including ID and name

## Configuration

- **Resource Group**: The Azure resource group containing the VM
- **VM Name**: The name of the virtual machine to restart

## Output

Returns the restarted VM information including:
- **id**: The Azure resource ID of the VM
- **name**: The name of the VM
- **resourceGroup**: The resource group containing the VM

## Notes

- The VM is rebooted in place — it keeps its compute allocation and IP addresses
- The operation is a Long-Running Operation (LRO) that typically takes 1-3 minutes
- The component waits for the VM to be fully restarted before completing
- This performs a graceful restart of the VM
`
}

func (c *RestartVMComponent) Icon() string {
	return "azure"
}

func (c *RestartVMComponent) Color() string {
	return "blue"
}

func (c *RestartVMComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":            "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
		"name":          "my-vm",
		"resourceGroup": "my-rg",
	}
}

func (c *RestartVMComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RestartVMComponent) Configuration() []configuration.Field {
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
			Description: "The name of the virtual machine to restart",
			Placeholder: "my-vm",
		},
	}
}

func (c *RestartVMComponent) Setup(ctx core.SetupContext) error {
	config := RestartVMConfiguration{}
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

func (c *RestartVMComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RestartVMComponent) Execute(ctx core.ExecutionContext) error {
	config := RestartVMConfiguration{}
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

	ctx.Logger.Infof("Restarting Azure VM: %s in resource group %s", config.Name, config.ResourceGroup)
	output, err := RestartVM(context.Background(), provider, req, ctx.Logger)
	if err != nil {
		return fmt.Errorf("failed to restart VM: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"azure.vm",
		[]any{output},
	)
}

func (c *RestartVMComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RestartVMComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *RestartVMComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *RestartVMComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RestartVMComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
