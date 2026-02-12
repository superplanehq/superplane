package azure

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
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

// CreateVMResponse contains created VM details.
type CreateVMResponse struct {
	VMID              string
	Name              string
	ProvisioningState string
	Location          string
	Size              string
	PublicIP          string
	PrivateIP         string
	AdminUsername     string
}

// CreateVM creates a VM and waits for provisioning completion.
func CreateVM(ctx context.Context, provider *AzureProvider, req CreateVMRequest, logger *logrus.Entry) (*CreateVMResponse, error) {
	if err := validateCreateVMRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	logger.Infof("Starting VM creation: name=%s, location=%s, size=%s",
		req.VMName, req.Location, req.Size)

	networkInterfaceID, err := ensureNetworkInterface(ctx, provider, req, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure network interface: %w", err)
	}

	osDiskType := armcompute.StorageAccountTypes(req.OSDiskType)
	if osDiskType == "" {
		osDiskType = armcompute.StorageAccountTypesStandardSSDLRS
	}

	var customData *string
	if strings.TrimSpace(req.CustomData) != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(req.CustomData))
		customData = to.Ptr(encoded)
	}

	vmParameters := armcompute.VirtualMachine{
		Location: to.Ptr(req.Location),
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: to.Ptr(armcompute.VirtualMachineSizeTypes(req.Size)),
			},
			StorageProfile: &armcompute.StorageProfile{
				ImageReference: &armcompute.ImageReference{
					Publisher: to.Ptr(req.ImagePublisher),
					Offer:     to.Ptr(req.ImageOffer),
					SKU:       to.Ptr(req.ImageSku),
					Version:   to.Ptr(req.ImageVersion),
				},
				OSDisk: &armcompute.OSDisk{
					CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesFromImage),
					ManagedDisk: &armcompute.ManagedDiskParameters{
						StorageAccountType: to.Ptr(osDiskType),
					},
				},
			},
			OSProfile: &armcompute.OSProfile{
				ComputerName:  to.Ptr(req.VMName),
				AdminUsername: to.Ptr(req.AdminUsername),
				AdminPassword: to.Ptr(req.AdminPassword),
				CustomData:    customData,
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
					{
						ID: to.Ptr(networkInterfaceID),
						Properties: &armcompute.NetworkInterfaceReferenceProperties{
							Primary: to.Ptr(true),
						},
					},
				},
			},
		},
	}

	logger.Infof("Initiating VM creation with Azure Compute API")

	client := provider.GetComputeClient()
	poller, err := client.BeginCreateOrUpdate(
		ctx,
		req.ResourceGroup,
		req.VMName,
		vmParameters,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to begin VM creation: %w", err)
	}

	logger.Infof("VM creation initiated, polling until completion...")

	result, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("VM creation failed during provisioning: %w", err)
	}

	vm := result.VirtualMachine
	if vm.ID == nil || vm.Name == nil {
		return nil, fmt.Errorf("invalid VM response: missing ID or Name")
	}

	provisioningState := "Unknown"
	if vm.Properties != nil && vm.Properties.ProvisioningState != nil {
		provisioningState = *vm.Properties.ProvisioningState
	}

	location := ""
	if vm.Location != nil {
		location = *vm.Location
	}

	size := ""
	if vm.Properties != nil && vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
		size = string(*vm.Properties.HardwareProfile.VMSize)
	}

	privateIP, publicIP, err := getPrimaryNICIPAddresses(ctx, provider, vm)
	if err != nil {
		logger.WithError(err).Warn("VM created but failed to resolve NIC IP addresses")
	}

	response := &CreateVMResponse{
		VMID:              *vm.ID,
		Name:              *vm.Name,
		ProvisioningState: provisioningState,
		Location:          location,
		Size:              size,
		PublicIP:          publicIP,
		PrivateIP:         privateIP,
		AdminUsername:     req.AdminUsername,
	}

	logger.Infof("VM created successfully: id=%s, state=%s", response.VMID, response.ProvisioningState)

	return response, nil
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

	subnetResp, err := provider.GetSubnetsClient().Get(ctx, req.ResourceGroup, virtualNetworkName, subnetName, nil)
	if err != nil {
		return "", fmt.Errorf("failed to resolve subnet %s in virtual network %s: %w", subnetName, virtualNetworkName, err)
	}
	if subnetResp.Subnet.ID == nil {
		return "", fmt.Errorf("resolved subnet %s in virtual network %s has no resource ID", subnetName, virtualNetworkName)
	}

	var publicIPAddress *armnetwork.PublicIPAddress
	if strings.TrimSpace(req.PublicIPName) != "" {
		publicIPID, err := ensurePublicIP(ctx, provider, req.ResourceGroup, req.Location, req.PublicIPName)
		if err != nil {
			return "", err
		}
		publicIPAddress = &armnetwork.PublicIPAddress{
			ID: to.Ptr(publicIPID),
		}
	}

	nicName := fmt.Sprintf("%s-nic", req.VMName)
	nicParams := armnetwork.Interface{
		Location: to.Ptr(req.Location),
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Name: to.Ptr("ipconfig1"),
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethod(IPAllocationMethodDynamic)),
						Subnet:                    &armnetwork.Subnet{ID: subnetResp.Subnet.ID},
						PublicIPAddress:           publicIPAddress,
					},
				},
			},
		},
	}

	logger.Infof("Creating network interface %s using subnet %s/%s", nicName, virtualNetworkName, subnetName)
	poller, err := provider.GetNetworkInterfacesClient().BeginCreateOrUpdate(ctx, req.ResourceGroup, nicName, nicParams, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create network interface %s: %w", nicName, err)
	}

	nicResult, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed while creating network interface %s: %w", nicName, err)
	}
	if nicResult.Interface.ID == nil {
		return "", fmt.Errorf("created network interface %s has no resource ID", nicName)
	}

	return *nicResult.Interface.ID, nil
}

func getPrimaryNICIPAddresses(ctx context.Context, provider *AzureProvider, vm armcompute.VirtualMachine) (string, string, error) {
	if vm.Properties == nil || vm.Properties.NetworkProfile == nil || len(vm.Properties.NetworkProfile.NetworkInterfaces) == 0 {
		return "", "", nil
	}

	primaryNIC := vm.Properties.NetworkProfile.NetworkInterfaces[0]
	if primaryNIC == nil || primaryNIC.ID == nil || strings.TrimSpace(*primaryNIC.ID) == "" {
		return "", "", nil
	}

	nicResourceGroup, nicName, err := resourceGroupAndNameFromResourceID(
		*primaryNIC.ID,
		fmt.Sprintf("%s/%s", ResourceProviderNetwork, "networkInterfaces"),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse primary NIC ID: %w", err)
	}

	nicResp, err := provider.GetNetworkInterfacesClient().Get(ctx, nicResourceGroup, nicName, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to get primary network interface %s: %w", nicName, err)
	}

	nic := nicResp.Interface
	if nic.Properties == nil || len(nic.Properties.IPConfigurations) == 0 || nic.Properties.IPConfigurations[0] == nil {
		return "", "", nil
	}

	ipConfig := nic.Properties.IPConfigurations[0]
	if ipConfig.Properties == nil {
		return "", "", nil
	}

	privateIP := ""
	if ipConfig.Properties.PrivateIPAddress != nil {
		privateIP = *ipConfig.Properties.PrivateIPAddress
	}

	publicIP := ""
	if ipConfig.Properties.PublicIPAddress != nil &&
		ipConfig.Properties.PublicIPAddress.ID != nil &&
		strings.TrimSpace(*ipConfig.Properties.PublicIPAddress.ID) != "" {
		publicIPResourceGroup, publicIPName, err := resourceGroupAndNameFromResourceID(
			*ipConfig.Properties.PublicIPAddress.ID,
			fmt.Sprintf("%s/%s", ResourceProviderNetwork, "publicIPAddresses"),
		)
		if err != nil {
			return privateIP, "", fmt.Errorf("failed to parse public IP resource ID: %w", err)
		}

		publicIPResp, err := provider.GetPublicIPClient().Get(ctx, publicIPResourceGroup, publicIPName, nil)
		if err != nil {
			return privateIP, "", fmt.Errorf("failed to get public IP resource %s: %w", publicIPName, err)
		}

		if publicIPResp.PublicIPAddress.Properties != nil && publicIPResp.PublicIPAddress.Properties.IPAddress != nil {
			publicIP = *publicIPResp.PublicIPAddress.Properties.IPAddress
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
	publicIPResp, err := provider.GetPublicIPClient().Get(ctx, resourceGroup, publicIPName, nil)
	if err == nil && publicIPResp.PublicIPAddress.ID != nil {
		return *publicIPResp.PublicIPAddress.ID, nil
	}
	if err != nil && !isAzureNotFound(err) {
		return "", fmt.Errorf("failed to get public IP %s: %w", publicIPName, err)
	}

	params := armnetwork.PublicIPAddress{
		Location: to.Ptr(location),
		SKU: &armnetwork.PublicIPAddressSKU{
			Name: to.Ptr(armnetwork.PublicIPAddressSKUName(SKUStandard)),
		},
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethod(IPAllocationMethodStatic)),
			PublicIPAddressVersion:   to.Ptr(armnetwork.IPVersionIPv4),
		},
	}

	poller, err := provider.GetPublicIPClient().BeginCreateOrUpdate(ctx, resourceGroup, publicIPName, params, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create public IP %s: %w", publicIPName, err)
	}
	publicIPResult, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed while creating public IP %s: %w", publicIPName, err)
	}
	if publicIPResult.PublicIPAddress.ID == nil {
		return "", fmt.Errorf("created public IP has no resource ID")
	}

	return *publicIPResult.PublicIPAddress.ID, nil
}

func isAzureNotFound(err error) bool {
	var responseErr *azcore.ResponseError
	if errors.As(err, &responseErr) {
		return strings.EqualFold(responseErr.ErrorCode, "NotFound") || strings.EqualFold(responseErr.ErrorCode, "ResourceNotFound")
	}
	normalized := strings.ToLower(err.Error())
	return strings.Contains(normalized, "notfound") || strings.Contains(normalized, "not found")
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
		req.OSDiskType != string(armcompute.StorageAccountTypesStandardLRS) &&
		req.OSDiskType != string(armcompute.StorageAccountTypesStandardSSDLRS) &&
		req.OSDiskType != string(armcompute.StorageAccountTypesPremiumLRS) {
		return fmt.Errorf("unsupported OS disk type: %s", req.OSDiskType)
	}

	return nil
}

const (
	VMSizeStandardB1s   = "Standard_B1s"
	VMSizeStandardB1ms  = "Standard_B1ms"
	VMSizeStandardB2s   = "Standard_B2s"
	VMSizeStandardD2sV3 = "Standard_D2s_v3"
	VMSizeStandardD4sV3 = "Standard_D4s_v3"
	VMSizeStandardD8sV3 = "Standard_D8s_v3"
	VMSizeStandardF2sV2 = "Standard_F2s_v2"
	VMSizeStandardF4sV2 = "Standard_F4s_v2"
	VMSizeStandardF8sV2 = "Standard_F8s_v2"
	VMSizeStandardE2sV3 = "Standard_E2s_v3"
	VMSizeStandardE4sV3 = "Standard_E4s_v3"
	VMSizeStandardE8sV3 = "Standard_E8s_v3"
)

var (
	ImageUbuntu1804LTS = ImageReference{
		Publisher: "Canonical",
		Offer:     "UbuntuServer",
		SKU:       "18.04-LTS",
		Version:   "latest",
	}

	ImageUbuntu2004LTS = ImageReference{
		Publisher: "Canonical",
		Offer:     "0001-com-ubuntu-server-focal",
		SKU:       "20_04-lts-gen2",
		Version:   "latest",
	}

	ImageWindowsServer2019 = ImageReference{
		Publisher: "MicrosoftWindowsServer",
		Offer:     "WindowsServer",
		SKU:       "2019-Datacenter",
		Version:   "latest",
	}

	ImageWindowsServer2022 = ImageReference{
		Publisher: "MicrosoftWindowsServer",
		Offer:     "WindowsServer",
		SKU:       "2022-datacenter-azure-edition",
		Version:   "latest",
	}
)

// ImageReference defines an image publisher/offer/sku/version tuple.
type ImageReference struct {
	Publisher string
	Offer     string
	SKU       string
	Version   string
}
