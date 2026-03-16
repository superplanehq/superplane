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

type CreateVMComponent struct {
	integration *AzureIntegration
}

type CreateVMConfiguration struct {
	ResourceGroup      string `json:"resourceGroup" mapstructure:"resourceGroup"`
	Name               string `json:"name" mapstructure:"name"`
	VirtualNetworkName string `json:"virtualNetworkName" mapstructure:"virtualNetworkName"`
	SubnetName         string `json:"subnetName" mapstructure:"subnetName"`
	Size               string `json:"size" mapstructure:"size"`
	PublicIPName       string `json:"publicIpName" mapstructure:"publicIpName"`
	AdminUsername      string `json:"adminUsername" mapstructure:"adminUsername"`
	AdminPassword      string `json:"adminPassword" mapstructure:"adminPassword"`
	NetworkInterfaceID string `json:"networkInterfaceId" mapstructure:"networkInterfaceId"`
	OSDiskType         string `json:"osDiskType" mapstructure:"osDiskType"`
	CustomData         string `json:"customData" mapstructure:"customData"`
	Image              string `json:"image" mapstructure:"image"`
}

func (c *CreateVMComponent) Name() string {
	return "azure.createVirtualMachine"
}

func (c *CreateVMComponent) Label() string {
	return "Create Virtual Machine"
}

func (c *CreateVMComponent) Description() string {
	return "Creates a new Azure Virtual Machine with the specified configuration"
}

func (c *CreateVMComponent) Documentation() string {
	return `
The Create Virtual Machine component creates a new Azure VM with full configuration options.

## Use Cases

- **Infrastructure provisioning**: Automatically create VMs as part of deployment workflows
- **Development environments**: Spin up temporary VMs for testing and development
- **Auto-scaling**: Create VMs in response to load or events
- **Disaster recovery**: Quickly provision replacement VMs

## How It Works

1. Validates the VM configuration parameters
2. Initiates VM creation via the Azure Compute API
3. Waits for the VM to be fully provisioned (using Azure's Long-Running Operation pattern)
4. Returns the VM details including ID, name, and provisioning state

## Configuration

- **Resource Group**: The Azure resource group where the VM will be created
- **Name**: The name for the new virtual machine
- **Location**: The Azure region (e.g., "eastus", "westeurope")
- **Size**: The VM size (e.g., "Standard_B1s", "Standard_D2s_v3")
- **Admin Username**: Administrator username for the VM
- **Admin Password**: Administrator password for the VM (must meet Azure complexity requirements)
- **Virtual Network / Subnet**: The network for the VM
- **Network Interface ID**: Optional existing NIC (overrides VNet/Subnet)
- **OS Image**: Select from common presets (Ubuntu, Debian, Windows Server)

## Output

Returns the created VM information including:
- **id**: The Azure resource ID of the VM
- **name**: The name of the VM
- **provisioningState**: The provisioning state (typically "Succeeded")
- **location**: The Azure region where the VM was created
- **size**: The VM size

## Notes

- The VM creation is a Long-Running Operation (LRO) that typically takes 2-5 minutes
- The component waits for the VM to be fully provisioned before completing
- The admin password must meet Azure's complexity requirements (12+ characters, mixed case, numbers, symbols)
- If Network Interface ID is empty, a NIC is created automatically from the selected VNet/Subnet
`
}

func (c *CreateVMComponent) Icon() string {
	return "azure"
}

func (c *CreateVMComponent) Color() string {
	return "blue"
}

func (c *CreateVMComponent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":                "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
		"name":              "my-vm",
		"provisioningState": "Succeeded",
		"location":          "eastus",
		"size":              "Standard_B1s",
	}
}

func (c *CreateVMComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateVMComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceGroup",
			Label:       "Resource Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Azure resource group where the VM will be created",
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
			Description: "The name for the new virtual machine",
			Placeholder: "my-vm",
		},
		{
			Name:        "size",
			Label:       "VM Size",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The size of the virtual machine",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeVMSizeDropdown,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "resourceGroup",
							ValueFrom: &configuration.ParameterValueFrom{Field: "resourceGroup"},
						},
					},
				},
			},
		},
		{
			Name:        "virtualNetworkName",
			Label:       "Virtual Network",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select an existing virtual network in the selected resource group",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeVirtualNetworkDropdown,
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "resourceGroup",
							ValueFrom: &configuration.ParameterValueFrom{Field: "resourceGroup"},
						},
					},
				},
			},
		},
		{
			Name:        "subnetName",
			Label:       "Subnet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select an existing subnet in the selected virtual network",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeSubnetDropdown,
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "resourceGroup",
							ValueFrom: &configuration.ParameterValueFrom{Field: "resourceGroup"},
						},
						{
							Name:      "virtualNetworkName",
							ValueFrom: &configuration.ParameterValueFrom{Field: "virtualNetworkName"},
						},
					},
				},
			},
		},
		{
			Name:        "publicIpName",
			Label:       "Public IP Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional public IP resource name. Leave empty for private-only VM",
			Placeholder: "my-vm-pip",
		},
		{
			Name:        "adminUsername",
			Label:       "Admin Username",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Administrator username for the VM",
			Placeholder: "azureuser",
		},
		{
			Name:        "adminPassword",
			Label:       "Admin Password",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Administrator password (12+ characters, mixed case, numbers, symbols)",
			Placeholder: "••••••••••••",
			Sensitive:   true,
		},
		{
			Name:        "networkInterfaceId",
			Label:       "Network Interface ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional: use an existing network interface instead of creating one from VNet/Subnet",
			Placeholder: "/subscriptions/.../resourceGroups/.../providers/Microsoft.Network/networkInterfaces/my-nic",
		},
		{
			Name:        "osDiskType",
			Label:       "OS Disk Type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "StandardSSD_LRS",
			Description: "Managed disk performance tier for the VM OS disk",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Standard HDD", Value: "Standard_LRS"},
						{Label: "Standard SSD", Value: "StandardSSD_LRS"},
						{Label: "Premium SSD", Value: "Premium_LRS"},
					},
				},
			},
		},
		{
			Name:        "customData",
			Label:       "Custom Data (cloud-init)",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Cloud-init script (bash/yaml) to run on boot",
			Placeholder: "#cloud-config\npackages:\n  - nginx",
		},
		{
			Name:        "image",
			Label:       "OS Image",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "The operating system image for the VM",
			Default:     "ubuntu-24.04",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Ubuntu 24.04 LTS", Value: "ubuntu-24.04"},
						{Label: "Ubuntu 22.04 LTS", Value: "ubuntu-22.04"},
						{Label: "Ubuntu 20.04 LTS", Value: "ubuntu-20.04"},
						{Label: "Debian 12", Value: "debian-12"},
						{Label: "Windows Server 2022", Value: "windows-2022"},
					},
				},
			},
		},
	}
}

func (c *CreateVMComponent) Setup(ctx core.SetupContext) error {
	config := CreateVMConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
	}
	if config.Name == "" {
		return fmt.Errorf("VM name is required")
	}
	if config.Size == "" {
		return fmt.Errorf("VM size is required")
	}
	if config.AdminUsername == "" {
		return fmt.Errorf("admin username is required")
	}
	if config.AdminPassword == "" {
		return fmt.Errorf("admin password is required")
	}
	if config.VirtualNetworkName == "" {
		return fmt.Errorf("virtual network is required")
	}
	if config.SubnetName == "" {
		return fmt.Errorf("subnet is required")
	}

	return nil
}

func (c *CreateVMComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateVMComponent) Execute(ctx core.ExecutionContext) error {
	config := CreateVMConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	provider, err := c.integration.ensureProvider(ctx.Integration)
	if err != nil {
		return fmt.Errorf("Azure provider not available: %w", err)
	}

	location, err := getResourceGroupLocation(context.Background(), provider, azureResourceName(config.ResourceGroup))
	if err != nil {
		return fmt.Errorf("failed to resolve location for resource group %s: %w", config.ResourceGroup, err)
	}

	image := ResolveImagePreset(config.Image)

	req := CreateVMRequest{
		ResourceGroup:      config.ResourceGroup,
		VMName:             config.Name,
		Location:           location,
		Size:               config.Size,
		PublicIPName:       config.PublicIPName,
		AdminUsername:      config.AdminUsername,
		AdminPassword:      config.AdminPassword,
		NetworkInterfaceID: config.NetworkInterfaceID,
		VirtualNetworkName: config.VirtualNetworkName,
		SubnetName:         config.SubnetName,
		OSDiskType:         config.OSDiskType,
		CustomData:         config.CustomData,
		ImagePublisher:     image.Publisher,
		ImageOffer:         image.Offer,
		ImageSku:           image.SKU,
		ImageVersion:       image.Version,
	}

	ctx.Logger.Infof("Creating Azure VM: %s in resource group %s", config.Name, config.ResourceGroup)
	output, err := CreateVM(context.Background(), provider, req, ctx.Logger)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"azure.vm",
		[]any{output},
	)
}

func (c *CreateVMComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateVMComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateVMComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for this component")
}

func (c *CreateVMComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateVMComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
