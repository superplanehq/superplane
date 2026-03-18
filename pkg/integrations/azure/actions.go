package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// CreateVMRequest contains parameters for Azure VM creation.
type CreateVMRequest struct {
	ResourceGroup      string
	VMName             string
	Location           string
	Size               string
	AdminUsername      string
	AdminPassword      string
	NetworkInterfaceID string
	VirtualNetworkName string
	SubnetName         string
	PublicIPName       string
	ImagePublisher     string
	ImageOffer         string
	ImageSku           string
	ImageVersion       string
	OSDiskType         string
	CustomData         string
}

// ARM REST response structs for unmarshaling

type armVM struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Location   string           `json:"location"`
	Properties *armVMProperties `json:"properties"`
}

type armVMProperties struct {
	ProvisioningState string              `json:"provisioningState"`
	HardwareProfile   *armHardwareProfile `json:"hardwareProfile"`
	NetworkProfile    *armNetworkProfile  `json:"networkProfile"`
	StorageProfile    *armStorageProfile  `json:"storageProfile"`
}

type armHardwareProfile struct {
	VMSize string `json:"vmSize"`
}

type armNetworkProfile struct {
	NetworkInterfaces []armNetworkInterfaceRef `json:"networkInterfaces"`
}

type armNetworkInterfaceRef struct {
	ID         string                       `json:"id"`
	Properties *armNetworkInterfaceRefProps `json:"properties,omitempty"`
}

type armNetworkInterfaceRefProps struct {
	DeleteOption string `json:"deleteOption,omitempty"`
}

type armStorageProfile struct {
	OsDisk    *armOsDisk    `json:"osDisk"`
	DataDisks []armDataDisk `json:"dataDisks"`
}

type armOsDisk struct {
	ManagedDisk  *armManagedDiskRef `json:"managedDisk"`
	DeleteOption string             `json:"deleteOption"`
}

type armDataDisk struct {
	Lun          int                `json:"lun"`
	ManagedDisk  *armManagedDiskRef `json:"managedDisk"`
	DeleteOption string             `json:"deleteOption"`
}

type armManagedDiskRef struct {
	ID string `json:"id"`
}

type armNIC struct {
	ID         string            `json:"id"`
	Location   string            `json:"location"`
	Properties *armNICProperties `json:"properties"`
}

type armNICProperties struct {
	IPConfigurations []armIPConfiguration `json:"ipConfigurations"`
}

type armIPConfiguration struct {
	Name       string                 `json:"name"`
	Properties *armIPConfigProperties `json:"properties"`
}

type armIPConfigProperties struct {
	PrivateIPAddress string             `json:"privateIPAddress"`
	PublicIPAddress  *armPublicIPRef    `json:"publicIPAddress"`
	Subnet           *armSubnetResponse `json:"subnet"`
}

type armPublicIPRef struct {
	ID         string               `json:"id"`
	Properties *armPublicIPRefProps `json:"properties,omitempty"`
}

type armPublicIPRefProps struct {
	DeleteOption string `json:"deleteOption,omitempty"`
}

type armPublicIP struct {
	ID         string                 `json:"id"`
	Properties *armPublicIPProperties `json:"properties"`
}

type armPublicIPProperties struct {
	IPAddress string `json:"ipAddress"`
}

type armSubnetResponse struct {
	ID string `json:"id"`
}

// CreateVM creates a VM and waits for provisioning completion.
func CreateVM(ctx context.Context, provider *AzureProvider, req CreateVMRequest, logger *logrus.Entry) (map[string]any, error) {
	if err := validateCreateVMRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	logger.Infof("Starting VM creation: name=%s, location=%s, size=%s",
		req.VMName, req.Location, req.Size)

	networkInterfaceID, err := ensureNetworkInterface(ctx, provider, req, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure network interface: %w", err)
	}

	osDiskType := req.OSDiskType
	if osDiskType == "" {
		osDiskType = "StandardSSD_LRS"
	}

	osProfile := map[string]any{
		"computerName":  req.VMName,
		"adminUsername": req.AdminUsername,
		"adminPassword": req.AdminPassword,
	}

	if strings.TrimSpace(req.CustomData) != "" {
		osProfile["customData"] = base64.StdEncoding.EncodeToString([]byte(req.CustomData))
	}

	vmBody := map[string]any{
		"location": req.Location,
		"properties": map[string]any{
			"hardwareProfile": map[string]any{
				"vmSize": req.Size,
			},
			"storageProfile": map[string]any{
				"imageReference": map[string]any{
					"publisher": req.ImagePublisher,
					"offer":     req.ImageOffer,
					"sku":       req.ImageSku,
					"version":   req.ImageVersion,
				},
				"osDisk": map[string]any{
					"createOption": "FromImage",
					"managedDisk": map[string]any{
						"storageAccountType": osDiskType,
					},
				},
			},
			"osProfile": osProfile,
			"networkProfile": map[string]any{
				"networkInterfaces": []map[string]any{
					{
						"id": networkInterfaceID,
						"properties": map[string]any{
							"primary": true,
						},
					},
				},
			},
		},
	}

	logger.Infof("Initiating VM creation with Azure Compute API")

	vmURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), req.ResourceGroup, req.VMName, armAPIVersionCompute)

	result, err := provider.getClient().putAndPoll(ctx, vmURL, vmBody)
	if err != nil {
		return nil, fmt.Errorf("VM creation failed: %w", err)
	}

	// Parse into the typed struct for validation and IP resolution.
	var vm armVM
	if err := json.Unmarshal(result, &vm); err != nil {
		return nil, fmt.Errorf("failed to parse VM response: %w", err)
	}

	if vm.ID == "" || vm.Name == "" {
		return nil, fmt.Errorf("invalid VM response: missing ID or Name")
	}

	// Unmarshal into a raw map so we emit the full, unmodified ARM response.
	var raw map[string]any
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse raw VM response: %w", err)
	}

	// Enrich with resolved IP addresses and the admin username (not in ARM response).
	privateIP, publicIP, err := getPrimaryNICIPAddresses(ctx, provider, vm)
	if err != nil {
		logger.WithError(err).Warn("VM created but failed to resolve NIC IP addresses")
	}

	raw["publicIp"] = publicIP
	raw["privateIp"] = privateIP
	raw["adminUsername"] = req.AdminUsername

	logger.Infof("VM created successfully: id=%s", vm.ID)

	return raw, nil
}

func ensureNetworkInterface(ctx context.Context, provider *AzureProvider, req CreateVMRequest, logger *logrus.Entry) (string, error) {
	if strings.TrimSpace(req.NetworkInterfaceID) != "" {
		return req.NetworkInterfaceID, nil
	}
	virtualNetworkName := azureResourceName(req.VirtualNetworkName)
	subnetName := azureResourceName(req.SubnetName)
	if virtualNetworkName == "" || subnetName == "" {
		return "", fmt.Errorf("either networkInterfaceId or (virtualNetworkName and subnetName) must be provided")
	}

	// Get subnet ID
	subnetURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), req.ResourceGroup, virtualNetworkName, subnetName, armAPIVersionNetwork)

	logger.Infof("resolving subnet: resourceGroup=%s vnet=%s subnet=%s", req.ResourceGroup, virtualNetworkName, subnetName)

	var subnet armSubnetResponse
	if err := provider.getClient().get(ctx, subnetURL, &subnet); err != nil {
		return "", fmt.Errorf("failed to resolve subnet %s in virtual network %s: %w", subnetName, virtualNetworkName, err)
	}
	if subnet.ID == "" {
		return "", fmt.Errorf("resolved subnet %s in virtual network %s has no resource ID", subnetName, virtualNetworkName)
	}

	logger.Infof("resolved subnet ID: %s", subnet.ID)

	// Optionally create public IP
	var publicIPAddress map[string]any
	if strings.TrimSpace(req.PublicIPName) != "" {
		publicIPID, err := ensurePublicIP(ctx, provider, req.ResourceGroup, req.Location, req.PublicIPName)
		if err != nil {
			return "", err
		}
		publicIPAddress = map[string]any{"id": publicIPID}
	}

	nicName := fmt.Sprintf("%s-nic", req.VMName)
	ipConfig := map[string]any{
		"privateIPAllocationMethod": "Dynamic",
		"subnet":                    map[string]any{"id": subnet.ID},
	}
	if publicIPAddress != nil {
		ipConfig["publicIPAddress"] = publicIPAddress
	}

	nicBody := map[string]any{
		"location": req.Location,
		"properties": map[string]any{
			"ipConfigurations": []map[string]any{
				{
					"name":       "ipconfig1",
					"properties": ipConfig,
				},
			},
		},
	}

	logger.Infof("Creating network interface %s using subnet %s/%s", nicName, virtualNetworkName, subnetName)

	nicURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), req.ResourceGroup, nicName, armAPIVersionNetwork)

	result, err := provider.getClient().putAndPoll(ctx, nicURL, nicBody)
	if err != nil {
		return "", fmt.Errorf("failed to create network interface %s: %w", nicName, err)
	}

	var nic armNIC
	if err := json.Unmarshal(result, &nic); err != nil {
		return "", fmt.Errorf("failed to parse NIC response: %w", err)
	}
	if nic.ID == "" {
		return "", fmt.Errorf("created network interface %s has no resource ID", nicName)
	}

	return nic.ID, nil
}

func getPrimaryNICIPAddresses(ctx context.Context, provider *AzureProvider, vm armVM) (string, string, error) {
	if vm.Properties == nil || vm.Properties.NetworkProfile == nil || len(vm.Properties.NetworkProfile.NetworkInterfaces) == 0 {
		return "", "", nil
	}

	primaryNICID := vm.Properties.NetworkProfile.NetworkInterfaces[0].ID
	if strings.TrimSpace(primaryNICID) == "" {
		return "", "", nil
	}

	nicResourceGroup, nicName, err := resourceGroupAndNameFromResourceID(primaryNICID, "Microsoft.Network/networkInterfaces")
	if err != nil {
		return "", "", fmt.Errorf("failed to parse primary NIC ID: %w", err)
	}

	nicURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), nicResourceGroup, nicName, armAPIVersionNetwork)

	var nic armNIC
	if err := provider.getClient().get(ctx, nicURL, &nic); err != nil {
		return "", "", fmt.Errorf("failed to get primary network interface %s: %w", nicName, err)
	}

	if nic.Properties == nil || len(nic.Properties.IPConfigurations) == 0 {
		return "", "", nil
	}

	ipConfig := nic.Properties.IPConfigurations[0]
	if ipConfig.Properties == nil {
		return "", "", nil
	}

	privateIP := ipConfig.Properties.PrivateIPAddress

	publicIP := ""
	if ipConfig.Properties.PublicIPAddress != nil && strings.TrimSpace(ipConfig.Properties.PublicIPAddress.ID) != "" {
		publicIPResourceGroup, publicIPName, err := resourceGroupAndNameFromResourceID(
			ipConfig.Properties.PublicIPAddress.ID,
			"Microsoft.Network/publicIPAddresses",
		)
		if err != nil {
			return privateIP, "", fmt.Errorf("failed to parse public IP resource ID: %w", err)
		}

		pipURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/%s?api-version=%s",
			armBaseURL, provider.GetSubscriptionID(), publicIPResourceGroup, publicIPName, armAPIVersionNetwork)

		var pip armPublicIP
		if err := provider.getClient().get(ctx, pipURL, &pip); err != nil {
			return privateIP, "", fmt.Errorf("failed to get public IP resource %s: %w", publicIPName, err)
		}

		if pip.Properties != nil {
			publicIP = pip.Properties.IPAddress
		}
	}

	return privateIP, publicIP, nil
}

func resourceGroupAndNameFromResourceID(resourceID, expectedResourceType string) (string, string, error) {
	trimmedID := strings.TrimSpace(resourceID)
	if trimmedID == "" {
		return "", "", fmt.Errorf("resource ID is empty")
	}

	normalizedID := strings.Trim(trimmedID, "/")
	parts := strings.Split(normalizedID, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid resource ID: %s", resourceID)
	}

	if expectedResourceType != "" {
		expectedPathToken := "/" + strings.ToLower(expectedResourceType) + "/"
		if !strings.Contains(strings.ToLower(trimmedID), expectedPathToken) {
			return "", "", fmt.Errorf("resource ID %q is not a %s resource", resourceID, expectedResourceType)
		}
	}

	resourceGroup := ""
	for i := 0; i < len(parts)-1; i++ {
		if strings.EqualFold(parts[i], "resourceGroups") {
			resourceGroup = parts[i+1]
			break
		}
	}
	if resourceGroup == "" {
		return "", "", fmt.Errorf("resource group segment not found in %q", resourceID)
	}

	resourceName := parts[len(parts)-1]
	if resourceName == "" {
		return "", "", fmt.Errorf("resource name segment not found in %q", resourceID)
	}

	return resourceGroup, resourceName, nil
}

func ensurePublicIP(ctx context.Context, provider *AzureProvider, resourceGroup, location, publicIPName string) (string, error) {
	pipURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/%s?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), resourceGroup, publicIPName, armAPIVersionNetwork)

	var existing armPublicIP
	err := provider.getClient().get(ctx, pipURL, &existing)
	if err == nil && existing.ID != "" {
		return existing.ID, nil
	}
	if err != nil && !isARMNotFound(err) {
		return "", fmt.Errorf("failed to get public IP %s: %w", publicIPName, err)
	}

	pipBody := map[string]any{
		"location": location,
		"sku": map[string]any{
			"name": "Standard",
		},
		"properties": map[string]any{
			"publicIPAllocationMethod": "Static",
			"publicIPAddressVersion":   "IPv4",
		},
	}

	result, err := provider.getClient().putAndPoll(ctx, pipURL, pipBody)
	if err != nil {
		return "", fmt.Errorf("failed to create public IP %s: %w", publicIPName, err)
	}

	var pip armPublicIP
	if err := json.Unmarshal(result, &pip); err != nil {
		return "", fmt.Errorf("failed to parse public IP response: %w", err)
	}
	if pip.ID == "" {
		return "", fmt.Errorf("created public IP has no resource ID")
	}

	return pip.ID, nil
}

// VMActionRequest contains parameters for Azure VM actions (start, stop, deallocate, restart).
type VMActionRequest struct {
	ResourceGroup string
	VMName        string
}

// validateVMActionRequest validates required VM action request fields.
func validateVMActionRequest(req VMActionRequest) error {
	if req.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
	}
	if req.VMName == "" {
		return fmt.Errorf("VM name is required")
	}
	return nil
}

// getVMRaw fetches the full VM resource and returns it as a raw map.
func getVMRaw(ctx context.Context, provider *AzureProvider, resourceGroup, vmName string) (map[string]any, error) {
	vmURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s?api-version=%s",
		provider.getClient().getBaseURL(), provider.GetSubscriptionID(), resourceGroup, vmName, armAPIVersionCompute)

	var raw map[string]any
	if err := provider.getClient().get(ctx, vmURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}

	return raw, nil
}

// StartVM starts a stopped/deallocated VM and waits for completion.
func StartVM(ctx context.Context, provider *AzureProvider, req VMActionRequest, logger *logrus.Entry) (map[string]any, error) {
	if err := validateVMActionRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	logger.Infof("Starting VM: name=%s, resourceGroup=%s", req.VMName, req.ResourceGroup)

	actionURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s/start?api-version=%s",
		provider.getClient().getBaseURL(), provider.GetSubscriptionID(), req.ResourceGroup, req.VMName, armAPIVersionCompute)

	if err := provider.getClient().postAndPoll(ctx, actionURL, nil); err != nil {
		return nil, fmt.Errorf("VM start failed: %w", err)
	}

	raw, err := getVMRaw(ctx, provider, req.ResourceGroup, req.VMName)
	if err != nil {
		return nil, fmt.Errorf("VM started but failed to fetch VM state: %w", err)
	}

	logger.Infof("VM started successfully: name=%s", req.VMName)
	return raw, nil
}

// StopVM powers off a VM (without deallocating) and waits for completion.
func StopVM(ctx context.Context, provider *AzureProvider, req VMActionRequest, logger *logrus.Entry) (map[string]any, error) {
	if err := validateVMActionRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	logger.Infof("Starting VM power-off: name=%s, resourceGroup=%s", req.VMName, req.ResourceGroup)

	actionURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s/powerOff?api-version=%s",
		provider.getClient().getBaseURL(), provider.GetSubscriptionID(), req.ResourceGroup, req.VMName, armAPIVersionCompute)

	if err := provider.getClient().postAndPoll(ctx, actionURL, nil); err != nil {
		return nil, fmt.Errorf("VM power-off failed: %w", err)
	}

	raw, err := getVMRaw(ctx, provider, req.ResourceGroup, req.VMName)
	if err != nil {
		return nil, fmt.Errorf("VM powered off but failed to fetch VM state: %w", err)
	}

	logger.Infof("VM powered off successfully: name=%s", req.VMName)
	return raw, nil
}

// DeallocateVM deallocates a VM (stop + release compute resources) and waits for completion.
func DeallocateVM(ctx context.Context, provider *AzureProvider, req VMActionRequest, logger *logrus.Entry) (map[string]any, error) {
	if err := validateVMActionRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	logger.Infof("Starting VM deallocation: name=%s, resourceGroup=%s", req.VMName, req.ResourceGroup)

	actionURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s/deallocate?api-version=%s",
		provider.getClient().getBaseURL(), provider.GetSubscriptionID(), req.ResourceGroup, req.VMName, armAPIVersionCompute)

	if err := provider.getClient().postAndPoll(ctx, actionURL, nil); err != nil {
		return nil, fmt.Errorf("VM deallocation failed: %w", err)
	}

	raw, err := getVMRaw(ctx, provider, req.ResourceGroup, req.VMName)
	if err != nil {
		return nil, fmt.Errorf("VM deallocated but failed to fetch VM state: %w", err)
	}

	logger.Infof("VM deallocated successfully: name=%s", req.VMName)
	return raw, nil
}

// RestartVM restarts a VM in place and waits for completion.
func RestartVM(ctx context.Context, provider *AzureProvider, req VMActionRequest, logger *logrus.Entry) (map[string]any, error) {
	if err := validateVMActionRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	logger.Infof("Starting VM restart: name=%s, resourceGroup=%s", req.VMName, req.ResourceGroup)

	actionURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s/restart?api-version=%s",
		provider.getClient().getBaseURL(), provider.GetSubscriptionID(), req.ResourceGroup, req.VMName, armAPIVersionCompute)

	if err := provider.getClient().postAndPoll(ctx, actionURL, nil); err != nil {
		return nil, fmt.Errorf("VM restart failed: %w", err)
	}

	raw, err := getVMRaw(ctx, provider, req.ResourceGroup, req.VMName)
	if err != nil {
		return nil, fmt.Errorf("VM restarted but failed to fetch VM state: %w", err)
	}

	logger.Infof("VM restarted successfully: name=%s", req.VMName)
	return raw, nil
}

// DeleteVMRequest contains parameters for Azure VM deletion.
type DeleteVMRequest struct {
	ResourceGroup             string
	VMName                    string
	DeleteAssociatedResources bool
}

// DeleteVM deletes a VM and waits for the operation to complete.
// If DeleteAssociatedResources is true, it first marks the VM's OS disk, data disks,
// NICs, and public IPs for cascade deletion using Azure's deleteOption mechanism.
func DeleteVM(ctx context.Context, provider *AzureProvider, req DeleteVMRequest, logger *logrus.Entry) (map[string]any, error) {
	if err := validateDeleteVMRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	logger.Infof("Starting VM deletion: name=%s, resourceGroup=%s, deleteAssociatedResources=%v",
		req.VMName, req.ResourceGroup, req.DeleteAssociatedResources)

	if req.DeleteAssociatedResources {
		if err := markVMResourcesForDeletion(ctx, provider, req.ResourceGroup, req.VMName, logger); err != nil {
			return nil, fmt.Errorf("failed to mark associated resources for deletion: %w", err)
		}
	}

	vmURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s?api-version=%s",
		provider.getClient().getBaseURL(), provider.GetSubscriptionID(), req.ResourceGroup, req.VMName, armAPIVersionCompute)

	if err := provider.getClient().deleteAndPoll(ctx, vmURL); err != nil {
		return nil, fmt.Errorf("VM deletion failed: %w", err)
	}

	vmID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s",
		provider.GetSubscriptionID(), req.ResourceGroup, req.VMName)

	response := map[string]any{
		"id":            vmID,
		"name":          req.VMName,
		"resourceGroup": req.ResourceGroup,
	}

	logger.Infof("VM deleted successfully: name=%s, id=%s", req.VMName, vmID)
	return response, nil
}

// markVMResourcesForDeletion sets deleteOption=Delete on the VM's associated resources
// (OS disk, data disks, NICs) and on each NIC's public IPs so that when the VM is deleted,
// Azure automatically cascade-deletes these resources.
func markVMResourcesForDeletion(ctx context.Context, provider *AzureProvider, resourceGroup, vmName string, logger *logrus.Entry) error {
	client := provider.getClient()
	subscriptionID := provider.GetSubscriptionID()

	// Step 1: GET the VM to discover associated resources.
	vmURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s?api-version=%s",
		client.getBaseURL(), subscriptionID, resourceGroup, vmName, armAPIVersionCompute)

	var vm armVM
	if err := client.get(ctx, vmURL, &vm); err != nil {
		return fmt.Errorf("failed to get VM: %w", err)
	}

	// Step 2: Build and send PATCH to set deleteOption on disks and NICs.
	vmPatch := buildVMDeleteOptionPatch(&vm)
	if vmPatch != nil {
		logger.Info("Marking VM disks and NICs for cascade deletion")
		resp, err := client.patch(ctx, vmURL, vmPatch)
		if err != nil {
			return fmt.Errorf("failed to patch VM delete options: %w", err)
		}
		resp.Body.Close()
	}

	// Step 3: For each NIC, mark its public IPs for cascade deletion.
	if vm.Properties != nil && vm.Properties.NetworkProfile != nil {
		for _, nicRef := range vm.Properties.NetworkProfile.NetworkInterfaces {
			if nicRef.ID == "" {
				continue
			}
			if err := markNICPublicIPsForDeletion(ctx, client, nicRef.ID, logger); err != nil {
				logger.Warnf("Failed to mark public IPs for deletion on NIC %s: %v", nicRef.ID, err)
			}
		}
	}

	return nil
}

// buildVMDeleteOptionPatch constructs the PATCH body to set deleteOption=Delete
// on the VM's OS disk, data disks, and network interfaces.
func buildVMDeleteOptionPatch(vm *armVM) map[string]any {
	if vm.Properties == nil {
		return nil
	}

	patch := map[string]any{}
	properties := map[string]any{}

	// Mark OS disk and data disks.
	if vm.Properties.StorageProfile != nil {
		storageProfile := map[string]any{}

		if vm.Properties.StorageProfile.OsDisk != nil {
			storageProfile["osDisk"] = map[string]any{
				"deleteOption": "Delete",
			}
		}

		if len(vm.Properties.StorageProfile.DataDisks) > 0 {
			dataDisks := make([]map[string]any, len(vm.Properties.StorageProfile.DataDisks))
			for i, disk := range vm.Properties.StorageProfile.DataDisks {
				dataDisks[i] = map[string]any{
					"lun":          disk.Lun,
					"deleteOption": "Delete",
				}
			}
			storageProfile["dataDisks"] = dataDisks
		}

		properties["storageProfile"] = storageProfile
	}

	// Mark NICs.
	if vm.Properties.NetworkProfile != nil && len(vm.Properties.NetworkProfile.NetworkInterfaces) > 0 {
		nics := make([]map[string]any, len(vm.Properties.NetworkProfile.NetworkInterfaces))
		for i, nic := range vm.Properties.NetworkProfile.NetworkInterfaces {
			nics[i] = map[string]any{
				"id": nic.ID,
				"properties": map[string]any{
					"deleteOption": "Delete",
				},
			}
		}
		properties["networkProfile"] = map[string]any{
			"networkInterfaces": nics,
		}
	}

	if len(properties) == 0 {
		return nil
	}

	patch["properties"] = properties
	return patch
}

// markNICPublicIPsForDeletion reads a NIC and patches it to set deleteOption=Delete
// on any public IP references, so they are cascade-deleted with the VM.
func markNICPublicIPsForDeletion(ctx context.Context, client *armClient, nicID string, logger *logrus.Entry) error {
	nicURL := fmt.Sprintf("%s%s?api-version=%s", client.getBaseURL(), nicID, armAPIVersionNetwork)

	var nic armNIC
	if err := client.get(ctx, nicURL, &nic); err != nil {
		return fmt.Errorf("failed to get NIC: %w", err)
	}

	if nic.Properties == nil {
		return nil
	}

	// Check if any ip configuration has a public IP.
	hasPublicIP := false
	for _, ipConfig := range nic.Properties.IPConfigurations {
		if ipConfig.Properties != nil && ipConfig.Properties.PublicIPAddress != nil && ipConfig.Properties.PublicIPAddress.ID != "" {
			hasPublicIP = true
			break
		}
	}

	if !hasPublicIP {
		return nil
	}

	// Build the PATCH body for the NIC, setting deleteOption on public IPs.
	ipConfigs := make([]map[string]any, len(nic.Properties.IPConfigurations))
	for i, ipConfig := range nic.Properties.IPConfigurations {
		ipConfigPatch := map[string]any{
			"name":       ipConfig.Name,
			"properties": map[string]any{},
		}

		props := ipConfigPatch["properties"].(map[string]any)

		// Subnet is required for NIC PATCH operations.
		if ipConfig.Properties != nil && ipConfig.Properties.Subnet != nil {
			props["subnet"] = map[string]any{"id": ipConfig.Properties.Subnet.ID}
		}

		if ipConfig.Properties != nil && ipConfig.Properties.PublicIPAddress != nil && ipConfig.Properties.PublicIPAddress.ID != "" {
			props["publicIPAddress"] = map[string]any{
				"id": ipConfig.Properties.PublicIPAddress.ID,
				"properties": map[string]any{
					"deleteOption": "Delete",
				},
			}
			logger.Infof("Marking public IP %s for cascade deletion", ipConfig.Properties.PublicIPAddress.ID)
		}

		ipConfigs[i] = ipConfigPatch
	}

	nicPatch := map[string]any{
		"location": nic.Location,
		"properties": map[string]any{
			"ipConfigurations": ipConfigs,
		},
	}

	logger.Infof("Patching NIC %s to mark public IPs for deletion", nicID)
	resp, err := client.patch(ctx, nicURL, nicPatch)
	if err != nil {
		return fmt.Errorf("failed to patch NIC: %w", err)
	}
	resp.Body.Close()

	return nil
}

// validateDeleteVMRequest validates required VM deletion request fields.
func validateDeleteVMRequest(req DeleteVMRequest) error {
	if req.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
	}
	if req.VMName == "" {
		return fmt.Errorf("VM name is required")
	}
	return nil
}

// validateCreateVMRequest validates required VM request fields.
func validateCreateVMRequest(req CreateVMRequest) error {
	if req.ResourceGroup == "" {
		return fmt.Errorf("resource group is required")
	}
	if req.VMName == "" {
		return fmt.Errorf("VM name is required")
	}
	if req.Location == "" {
		return fmt.Errorf("location is required")
	}
	if req.Size == "" {
		return fmt.Errorf("VM size is required")
	}
	if req.AdminUsername == "" {
		return fmt.Errorf("admin username is required")
	}
	if req.AdminPassword == "" {
		return fmt.Errorf("admin password is required")
	}

	if strings.TrimSpace(req.NetworkInterfaceID) == "" &&
		(strings.TrimSpace(req.VirtualNetworkName) == "" || strings.TrimSpace(req.SubnetName) == "") {
		return fmt.Errorf("either networkInterfaceId or (virtualNetworkName and subnetName) is required")
	}

	if req.ImagePublisher == "" {
		return fmt.Errorf("image publisher is required")
	}
	if req.ImageOffer == "" {
		return fmt.Errorf("image offer is required")
	}
	if req.ImageSku == "" {
		return fmt.Errorf("image SKU is required")
	}
	if req.ImageVersion == "" {
		return fmt.Errorf("image version is required")
	}

	if req.OSDiskType != "" &&
		req.OSDiskType != "Standard_LRS" &&
		req.OSDiskType != "StandardSSD_LRS" &&
		req.OSDiskType != "Premium_LRS" {
		return fmt.Errorf("unsupported OS disk type: %s", req.OSDiskType)
	}

	return nil
}

// ImageReference defines an image publisher/offer/sku/version tuple.
type ImageReference struct {
	Publisher string
	Offer     string
	SKU       string
	Version   string
}

var ImageUbuntu2004LTS = ImageReference{
	Publisher: "Canonical",
	Offer:     "0001-com-ubuntu-server-focal",
	SKU:       "20_04-lts-gen2",
	Version:   "latest",
}

var ImageUbuntu2204LTS = ImageReference{
	Publisher: "Canonical",
	Offer:     "0001-com-ubuntu-server-jammy",
	SKU:       "22_04-lts-gen2",
	Version:   "latest",
}

var ImageUbuntu2404LTS = ImageReference{
	Publisher: "Canonical",
	Offer:     "ubuntu-24_04-lts",
	SKU:       "server",
	Version:   "latest",
}

var ImageDebian12 = ImageReference{
	Publisher: "Debian",
	Offer:     "debian-12",
	SKU:       "12-gen2",
	Version:   "latest",
}

var ImageWindowsServer2022 = ImageReference{
	Publisher: "MicrosoftWindowsServer",
	Offer:     "WindowsServer",
	SKU:       "2022-datacenter-g2",
	Version:   "latest",
}

// ImagePresets maps preset keys to their image references.
var ImagePresets = map[string]ImageReference{
	"ubuntu-24.04": ImageUbuntu2404LTS,
	"ubuntu-22.04": ImageUbuntu2204LTS,
	"ubuntu-20.04": ImageUbuntu2004LTS,
	"debian-12":    ImageDebian12,
	"windows-2022": ImageWindowsServer2022,
}

// ResolveImagePreset returns the ImageReference for a preset key,
// or the Ubuntu 24.04 default if the key is empty or unknown.
func ResolveImagePreset(preset string) ImageReference {
	if img, ok := ImagePresets[preset]; ok {
		return img
	}
	return ImageUbuntu2404LTS
}
