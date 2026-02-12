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

type CreateVMComponent struct{}

type CreateVMConfiguration struct {
	ResourceGroup      string `json:"resourceGroup" mapstructure:"resourceGroup"`
	Name               string `json:"name" mapstructure:"name"`
	Location           string `json:"location" mapstructure:"location"`
	VirtualNetworkName string `json:"virtualNetworkName" mapstructure:"virtualNetworkName"`
	SubnetName         string `json:"subnetName" mapstructure:"subnetName"`
	Size               string `json:"size" mapstructure:"size"`
	PublicIPName       string `json:"publicIpName" mapstructure:"publicIpName"`
	AdminUsername      string `json:"adminUsername" mapstructure:"adminUsername"`
	AdminPassword      string `json:"adminPassword" mapstructure:"adminPassword"`
	NetworkInterfaceID string `json:"networkInterfaceId" mapstructure:"networkInterfaceId"`
	OSDiskType         string `json:"osDiskType" mapstructure:"osDiskType"`
	CustomData         string `json:"customData" mapstructure:"customData"`
	ImagePublisher     string `json:"imagePublisher" mapstructure:"imagePublisher"`
	ImageOffer         string `json:"imageOffer" mapstructure:"imageOffer"`
	ImageSku           string `json:"imageSku" mapstructure:"imageSku"`
	ImageVersion       string `json:"imageVersion" mapstructure:"imageVersion"`
}

func (c *CreateVMComponent) Name() string {
	return "azure.createVirtualMachine"
}

func (c *CreateVMComponent) Label() string {
	return "Azure • Create Virtual Machine"
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
- **Network Interface ID**: Optional existing NIC. Leave empty to create NIC from selected VNet/Subnet.
- **Image**: The OS image to use (publisher, offer, SKU, version)

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
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The Azure region where the VM will be created",
			Default:     "eastus",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "East US", Value: "eastus"},
						{Label: "East US 2", Value: "eastus2"},
						{Label: "West US", Value: "westus"},
						{Label: "West US 2", Value: "westus2"},
						{Label: "West US 3", Value: "westus3"},
						{Label: "Central US", Value: "centralus"},
						{Label: "North Central US", Value: "northcentralus"},
						{Label: "South Central US", Value: "southcentralus"},
						{Label: "West Central US", Value: "westcentralus"},
						{Label: "North Europe", Value: "northeurope"},
						{Label: "West Europe", Value: "westeurope"},
						{Label: "UK South", Value: "uksouth"},
						{Label: "UK West", Value: "ukwest"},
						{Label: "Southeast Asia", Value: "southeastasia"},
						{Label: "East Asia", Value: "eastasia"},
						{Label: "Australia East", Value: "australiaeast"},
						{Label: "Australia Southeast", Value: "australiasoutheast"},
						{Label: "Japan East", Value: "japaneast"},
						{Label: "Japan West", Value: "japanwest"},
						{Label: "Brazil South", Value: "brazilsouth"},
						{Label: "Canada Central", Value: "canadacentral"},
						{Label: "Canada East", Value: "canadaeast"},
						{Label: "France Central", Value: "francecentral"},
						{Label: "Germany West Central", Value: "germanywestcentral"},
						{Label: "Korea Central", Value: "koreacentral"},
						{Label: "Central India", Value: "centralindia"},
						{Label: "South India", Value: "southindia"},
						{Label: "West India", Value: "westindia"},
						{Label: "Poland Central", Value: "polandcentral"},
						{Label: "Austria East", Value: "austriaeast"},
						{Label: "UAE North", Value: "uaenorth"},
						{Label: "Switzerland North", Value: "switzerlandnorth"},
						{Label: "Spain Central", Value: "spaincentral"},
					},
				},
			},
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
							Name:      "location",
							ValueFrom: &configuration.ParameterValueFrom{Field: "location"},
						},
					},
				},
			},
		},
		{
			Name:        "virtualNetworkName",
			Label:       "Virtual Network",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
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
			Required:    false,
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
			Description: "Optional: existing network interface ID (leave empty to create NIC from selected VNet/Subnet)",
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
			Name:        "imagePublisher",
			Label:       "Image Publisher",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The publisher of the VM image",
			Default:     ImageUbuntu2004LTS.Publisher,
			Placeholder: "Canonical",
		},
		{
			Name:        "imageOffer",
			Label:       "Image Offer",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The offer of the VM image",
			Default:     ImageUbuntu2004LTS.Offer,
			Placeholder: "UbuntuServer",
		},
		{
			Name:        "imageSku",
			Label:       "Image SKU",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The SKU of the VM image",
			Default:     ImageUbuntu2004LTS.SKU,
			Placeholder: "20.04-LTS",
		},
		{
			Name:        "imageVersion",
			Label:       "Image Version",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The version of the VM image",
			Default:     ImageUbuntu2004LTS.Version,
			Placeholder: "latest",
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
	if config.Location == "" {
		return fmt.Errorf("location is required")
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
	if config.NetworkInterfaceID == "" && (config.VirtualNetworkName == "" || config.SubnetName == "") {
		return fmt.Errorf("either network interface ID or both virtual network and subnet are required")
	}
	if config.OSDiskType == "" {
		config.OSDiskType = "StandardSSD_LRS"
	}

	if config.ImagePublisher == "" {
		config.ImagePublisher = ImageUbuntu2004LTS.Publisher
	}
	if config.ImageOffer == "" {
		config.ImageOffer = ImageUbuntu2004LTS.Offer
	}
	if config.ImageSku == "" {
		config.ImageSku = ImageUbuntu2004LTS.SKU
	}
	if config.ImageVersion == "" {
		config.ImageVersion = ImageUbuntu2004LTS.Version
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

	if ctx.Integration == nil {
		return fmt.Errorf("integration context is required")
	}

	tenantID, err := ctx.Integration.GetConfig("tenantId")
	if err != nil {
		return fmt.Errorf("failed to get tenant ID: %w", err)
	}

	clientID, err := ctx.Integration.GetConfig("clientId")
	if err != nil {
		return fmt.Errorf("failed to get client ID: %w", err)
	}

	subscriptionID, err := ctx.Integration.GetConfig("subscriptionId")
	if err != nil {
		return fmt.Errorf("failed to get subscription ID: %w", err)
	}

	provider, err := NewAzureProvider(
		context.Background(),
		string(tenantID),
		string(clientID),
		string(subscriptionID),
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create Azure provider: %w", err)
	}

	if config.ImagePublisher == "" {
		config.ImagePublisher = ImageUbuntu2004LTS.Publisher
	}
	if config.ImageOffer == "" {
		config.ImageOffer = ImageUbuntu2004LTS.Offer
	}
	if config.ImageSku == "" {
		config.ImageSku = ImageUbuntu2004LTS.SKU
	}
	if config.ImageVersion == "" {
		config.ImageVersion = ImageUbuntu2004LTS.Version
	}

	req := CreateVMRequest{
		ResourceGroup:      config.ResourceGroup,
		VMName:             config.Name,
		Location:           config.Location,
		Size:               config.Size,
		PublicIPName:       config.PublicIPName,
		AdminUsername:      config.AdminUsername,
		AdminPassword:      config.AdminPassword,
		NetworkInterfaceID: config.NetworkInterfaceID,
		VirtualNetworkName: config.VirtualNetworkName,
		SubnetName:         config.SubnetName,
		OSDiskType:         config.OSDiskType,
		CustomData:         config.CustomData,
		ImagePublisher:     config.ImagePublisher,
		ImageOffer:         config.ImageOffer,
		ImageSku:           config.ImageSku,
		ImageVersion:       config.ImageVersion,
	}

	ctx.Logger.Infof("Creating Azure VM: %s in resource group %s", config.Name, config.ResourceGroup)
	response, err := CreateVM(context.Background(), provider, req, ctx.Logger)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	output := map[string]any{
		"id":                response.VMID,
		"name":              response.Name,
		"provisioningState": response.ProvisioningState,
		"location":          response.Location,
		"size":              response.Size,
		"publicIp":          response.PublicIP,
		"privateIp":         response.PrivateIP,
		"adminUsername":     response.AdminUsername,
	}

	ctx.Logger.Infof("VM created successfully: %s (ID: %s)", response.Name, response.VMID)

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

func (c *CreateVMComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateVMComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}
