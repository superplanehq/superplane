package compute

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/azure/common"
)

const payloadTypeVMCreated = "azure.compute.vmCreated"

type CreateVirtualMachine struct{}

type CreateVMConfiguration struct {
	ResourceGroup      string          `json:"resourceGroup" mapstructure:"resourceGroup"`
	Name                string          `json:"name" mapstructure:"name"`
	Location            string          `json:"location" mapstructure:"location"`
	Size                string          `json:"size" mapstructure:"size"`
	Image               *VMImageRef     `json:"image" mapstructure:"image"`
	AdminUsername       string          `json:"adminUsername" mapstructure:"adminUsername"`
	AdminPassword       string          `json:"adminPassword" mapstructure:"adminPassword"`
	SSHPublicKey        string          `json:"sshPublicKey" mapstructure:"sshPublicKey"`
	OSDisk              *OSDiskConfig   `json:"osDisk" mapstructure:"osDisk"`
	Tags                []common.Tag    `json:"tags" mapstructure:"tags"`
	NetworkInterfaceID   string          `json:"networkInterfaceId" mapstructure:"networkInterfaceId"`
}

type VMImageRef struct {
	Publisher string `json:"publisher" mapstructure:"publisher"`
	Offer     string `json:"offer" mapstructure:"offer"`
	SKU       string `json:"sku" mapstructure:"sku"`
	Version   string `json:"version" mapstructure:"version"`
}

type OSDiskConfig struct {
	DiskSizeGB         int    `json:"diskSizeGB" mapstructure:"diskSizeGB"`
	StorageAccountType string `json:"storageAccountType" mapstructure:"storageAccountType"`
}

type CreateVMMetadata struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	Name          string `json:"name" mapstructure:"name"`
	Location      string `json:"location" mapstructure:"location"`
	Size          string `json:"size" mapstructure:"size"`
}

func (c *CreateVirtualMachine) Name() string {
	return "azure.compute.createVirtualMachine"
}

func (c *CreateVirtualMachine) Label() string {
	return "Azure Compute • Create Virtual Machine"
}

func (c *CreateVirtualMachine) Description() string {
	return "Create an Azure Virtual Machine in the specified resource group with the given size, image, and optional network/disk configuration."
}

func (c *CreateVirtualMachine) Documentation() string {
	return `
The Create Virtual Machine component provisions an Azure VM via the ARM API.

## Use Cases

- **Ephemeral workloads**: Spin up VMs from workflows for batch or one-off jobs
- **Environment provisioning**: Create dev/test VMs on demand
- **Scaling**: Add compute capacity when triggered by events

## Configuration

- **Resource group**: Must already exist
- **Name**: VM name (Azure naming rules)
- **Size**: e.g. Standard_B2s, Standard_D2s_v3
- **Image**: publisher, offer, sku, version (e.g. Canonical, 0001-com-ubuntu-server-jammy, 22_04-lts, latest)
- **Admin credentials**: Either password or SSH public key for Linux
- **Network interface**: Full ARM resource ID of an existing NIC (create a NIC in the same resource group first if needed)

## Output

Returns the VM resource ID, name, provisioning state, location, and size.
`
}

func (c *CreateVirtualMachine) Icon() string {
	return "azure"
}

func (c *CreateVirtualMachine) Color() string {
	return "blue"
}

func (c *CreateVirtualMachine) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateVirtualMachine) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceGroup",
			Label:       "Resource group",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the resource group (must exist)",
		},
		{
			Name:        "name",
			Label:       "VM name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the virtual machine",
		},
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Azure region (overrides integration default)",
		},
		{
			Name:        "size",
			Label:       "VM size",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. Standard_B2s, Standard_D2s_v3",
			Description: "Azure VM size",
		},
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeObject,
			Required:    true,
			Description: "OS image: publisher, offer, sku, version",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{Name: "publisher", Label: "Publisher", Type: configuration.FieldTypeString, Required: true},
						{Name: "offer", Label: "Offer", Type: configuration.FieldTypeString, Required: true},
						{Name: "sku", Label: "SKU", Type: configuration.FieldTypeString, Required: true},
						{Name: "version", Label: "Version", Type: configuration.FieldTypeString, Required: false},
					},
				},
			},
		},
		{
			Name:        "adminUsername",
			Label:       "Admin username",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Admin user name for the VM",
		},
		{
			Name:        "adminPassword",
			Label:       "Admin password",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Admin password (for Linux use SSH key instead if possible)",
		},
		{
			Name:        "sshPublicKey",
			Label:       "SSH public key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "SSH public key for Linux (alternative to password)",
		},
		{
			Name:        "osDisk",
			Label:       "OS disk",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "OS disk size and type",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{Name: "diskSizeGB", Label: "Size (GB)", Type: configuration.FieldTypeNumber, Required: false},
						{Name: "storageAccountType", Label: "Storage type", Type: configuration.FieldTypeString, Required: false},
					},
				},
			},
		},
		{
			Name:        "networkInterfaceId",
			Label:       "Network interface ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Full ARM resource ID of the network interface (e.g. /subscriptions/.../networkInterfaces/...)",
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "key", Label: "Key", Type: configuration.FieldTypeString, Required: true},
							{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true},
						},
					},
				},
			},
		},
	}
}

func (c *CreateVirtualMachine) Setup(ctx core.SetupContext) error {
	config := CreateVMConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.ResourceGroup) == "" {
		return fmt.Errorf("resource group is required")
	}
	if strings.TrimSpace(config.Name) == "" {
		return fmt.Errorf("VM name is required")
	}
	if strings.TrimSpace(config.Size) == "" {
		return fmt.Errorf("VM size is required")
	}
	if config.Image == nil || strings.TrimSpace(config.Image.Publisher) == "" || strings.TrimSpace(config.Image.Offer) == "" || strings.TrimSpace(config.Image.SKU) == "" {
		return fmt.Errorf("image (publisher, offer, sku) is required")
	}
	if strings.TrimSpace(config.AdminUsername) == "" {
		return fmt.Errorf("admin username is required")
	}
	if strings.TrimSpace(config.NetworkInterfaceID) == "" {
		return fmt.Errorf("network interface ID is required")
	}

	location := strings.TrimSpace(config.Location)
	if location == "" {
		location = common.LocationFromInstallation(ctx.Integration)
	}
	if location == "" {
		location = "eastus"
	}

	return ctx.Metadata.Set(CreateVMMetadata{
		ResourceGroup: strings.TrimSpace(config.ResourceGroup),
		Name:          strings.TrimSpace(config.Name),
		Location:      location,
		Size:          strings.TrimSpace(config.Size),
	})
}

func (c *CreateVirtualMachine) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateVirtualMachine) Execute(ctx core.ExecutionContext) error {
	config := CreateVMConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadata := CreateVMMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return err
	}

	subscriptionID := common.SubscriptionIDFromInstallation(ctx.Integration)
	if strings.TrimSpace(subscriptionID) == "" {
		return fmt.Errorf("subscription ID is required; check integration configuration")
	}

	client := NewClient(ctx.HTTP, creds, subscriptionID)

	params, err := c.buildCreateParams(&config, metadata.Location)
	if err != nil {
		return err
	}

	vm, err := client.CreateOrUpdateVM(metadata.ResourceGroup, metadata.Name, params)
	if err != nil {
		return err
	}

	output := map[string]any{
		"id":                vm.ID,
		"name":              vm.Name,
		"location":          vm.Location,
		"provisioningState": "",
		"vmSize":            metadata.Size,
	}
	if vm.Properties != nil {
		output["provisioningState"] = vm.Properties.ProvisioningState
		if vm.Properties.HardwareProfile != nil {
			output["vmSize"] = vm.Properties.HardwareProfile.VMSize
		}
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, payloadTypeVMCreated, []any{output})
}

func (c *CreateVirtualMachine) buildCreateParams(config *CreateVMConfiguration, location string) (*CreateOrUpdateVMParams, error) {
	imageRef := ImageReference{
		Publisher: strings.TrimSpace(config.Image.Publisher),
		Offer:     strings.TrimSpace(config.Image.Offer),
		SKU:       strings.TrimSpace(config.Image.SKU),
		Version:   strings.TrimSpace(config.Image.Version),
	}
	if imageRef.Version == "" {
		imageRef.Version = "latest"
	}

	osDisk := OSDisk{
		CreateOption: "FromImage",
		Caching:      "ReadWrite",
		ManagedDisk:  &ManagedDiskOptions{StorageAccountType: "Premium_LRS"},
		DiskSizeGB:   30,
	}
	if config.OSDisk != nil {
		if config.OSDisk.DiskSizeGB > 0 {
			osDisk.DiskSizeGB = config.OSDisk.DiskSizeGB
		}
		if strings.TrimSpace(config.OSDisk.StorageAccountType) != "" {
			osDisk.ManagedDisk = &ManagedDiskOptions{StorageAccountType: config.OSDisk.StorageAccountType}
		}
	}

	osProfile := OSProfile{
		ComputerName:  config.Name,
		AdminUsername: config.AdminUsername,
	}

	useSSH := strings.TrimSpace(config.SSHPublicKey) != ""
	if useSSH {
		osProfile.LinuxConfiguration = &LinuxConfiguration{
			DisablePasswordAuthentication: true,
			SSH: &SSHConfig{
				PublicKeys: []SSHPublicKey{{
					Path:    fmt.Sprintf("/home/%s/.ssh/authorized_keys", config.AdminUsername),
					KeyData: config.SSHPublicKey,
				}},
			},
		}
	} else {
		osProfile.AdminPassword = config.AdminPassword
	}

	params := &CreateOrUpdateVMParams{
		Location: location,
		Properties: CreateOrUpdateVMProps{
			HardwareProfile: HardwareProfile{VMSize: strings.TrimSpace(config.Size)},
			StorageProfile: StorageProfile{
				ImageReference: imageRef,
				OSDisk:         osDisk,
			},
			OSProfile: osProfile,
			NetworkProfile: NetworkProfile{
				NetworkInterfaces: []NetworkInterfaceRef{{ID: strings.TrimSpace(config.NetworkInterfaceID)}},
			},
		},
	}

	if len(config.Tags) > 0 {
		params.Tags = make(map[string]string)
		for _, t := range common.NormalizeTags(config.Tags) {
			params.Tags[t.Key] = t.Value
		}
	}

	return params, nil
}

func (c *CreateVirtualMachine) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateVirtualMachine) Actions() []core.Action {
	return nil
}

func (c *CreateVirtualMachine) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateVirtualMachine) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateVirtualMachine) Cleanup(ctx core.SetupContext) error {
	return nil
}
