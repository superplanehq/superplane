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

type StartVMComponent struct {
	integration *AzureIntegration
}

type StartVMConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	Name          string `json:"name" mapstructure:"name"`
}

func (c *StartVMComponent) Name() string {
	return "azure.startVirtualMachine"
}

func (c *StartVMComponent) Label() string {
	return "Start Virtual Machine"
}

func (c *StartVMComponent) Description() string {
	return "Starts a stopped or deallocated Azure Virtual Machine"
}

func (c *StartVMComponent) Documentation() string {
	return `
The Start Virtual Machine component starts a stopped or deallocated Azure VM.

## Use Cases

- **Resume workloads**: Start VMs that were previously stopped or deallocated
- **Scheduled startup**: Start VMs as part of scheduled workflows
- **Auto-scaling**: Start pre-provisioned VMs to handle increased demand
- **Disaster recovery**: Start standby VMs as part of failover workflows

## How It Works

1. Validates the VM parameters
2. Initiates VM start via the Azure Compute API
3. Waits for the VM to be fully started (using Azure's Long-Running Operation pattern)
4. Returns the VM details including ID and name

## Configuration

- **Resource Group**: The Azure resource group containing the VM
- **VM Name**: The name of the virtual machine to start

## Output

Returns the started VM information including:
- **id**: The Azure resource ID of the VM
- **name**: The name of the VM
- **resourceGroup**: The resource group containing the VM

## Notes

- The VM must be in a stopped or deallocated state to be started
- The operation is a Long-Running Operation (LRO) that typically takes 1-3 minutes
- The component waits for the VM to be fully started before completing
- Once started, the VM will begin incurring compute charges
`
}

func (c *StartVMComponent) Icon() string {
	return "azure"
}

func (c *StartVMComponent) Color() string {
	return "blue"
}

func (c *StartVMComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":            "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
		"name":          "my-vm",
		"resourceGroup": "my-rg",
	}
}

func (c *StartVMComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *StartVMComponent) Configuration() []configuration.Field {
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
			Description: "The name of the virtual machine to start",
			Placeholder: "my-vm",
		},
	}
}

func (c *StartVMComponent) Setup(ctx core.SetupContext) error {
	config := StartVMConfiguration{}
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

func (c *StartVMComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *StartVMComponent) Execute(ctx core.ExecutionContext) error {
	config := StartVMConfiguration{}
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

	ctx.Logger.Infof("Starting Azure VM: %s in resource group %s", config.Name, config.ResourceGroup)
	output, err := StartVM(context.Background(), provider, req, ctx.Logger)
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"azure.vm",
		[]any{output},
	)
}

func (c *StartVMComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *StartVMComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *StartVMComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *StartVMComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *StartVMComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
