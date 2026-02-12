package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCreateVMRequest(t *testing.T) {
	validRequest := CreateVMRequest{
		ResourceGroup:      "test-rg",
		VMName:             "test-vm",
		Location:           "eastus",
		Size:               "Standard_B1s",
		AdminUsername:      "azureuser",
		AdminPassword:      "P@ssw0rd123!",
		NetworkInterfaceID: "/subscriptions/test/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic",
		ImagePublisher:     "Canonical",
		ImageOffer:         "UbuntuServer",
		ImageSku:           "18.04-LTS",
		ImageVersion:       "latest",
	}

	tests := []struct {
		name        string
		modifyReq   func(*CreateVMRequest)
		expectedErr string
	}{
		{
			name:        "valid request",
			modifyReq:   func(r *CreateVMRequest) {},
			expectedErr: "",
		},
		{
			name: "missing resource group",
			modifyReq: func(r *CreateVMRequest) {
				r.ResourceGroup = ""
			},
			expectedErr: "resource group is required",
		},
		{
			name: "missing VM name",
			modifyReq: func(r *CreateVMRequest) {
				r.VMName = ""
			},
			expectedErr: "VM name is required",
		},
		{
			name: "missing location",
			modifyReq: func(r *CreateVMRequest) {
				r.Location = ""
			},
			expectedErr: "location is required",
		},
		{
			name: "missing VM size",
			modifyReq: func(r *CreateVMRequest) {
				r.Size = ""
			},
			expectedErr: "VM size is required",
		},
		{
			name: "missing admin username",
			modifyReq: func(r *CreateVMRequest) {
				r.AdminUsername = ""
			},
			expectedErr: "admin username is required",
		},
		{
			name: "missing admin password",
			modifyReq: func(r *CreateVMRequest) {
				r.AdminPassword = ""
			},
			expectedErr: "admin password is required",
		},
		{
			name: "missing network interface ID",
			modifyReq: func(r *CreateVMRequest) {
				r.NetworkInterfaceID = ""
			},
			expectedErr: "either networkInterfaceId or (virtualNetworkName and subnetName) is required",
		},
		{
			name: "valid request with implicit NIC fields",
			modifyReq: func(r *CreateVMRequest) {
				r.NetworkInterfaceID = ""
				r.VirtualNetworkName = "my-vnet"
				r.SubnetName = "my-subnet"
			},
			expectedErr: "",
		},
		{
			name: "missing image publisher",
			modifyReq: func(r *CreateVMRequest) {
				r.ImagePublisher = ""
			},
			expectedErr: "image publisher is required",
		},
		{
			name: "missing image offer",
			modifyReq: func(r *CreateVMRequest) {
				r.ImageOffer = ""
			},
			expectedErr: "image offer is required",
		},
		{
			name: "missing image SKU",
			modifyReq: func(r *CreateVMRequest) {
				r.ImageSku = ""
			},
			expectedErr: "image SKU is required",
		},
		{
			name: "missing image version",
			modifyReq: func(r *CreateVMRequest) {
				r.ImageVersion = ""
			},
			expectedErr: "image version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest
			tt.modifyReq(&req)

			err := validateCreateVMRequest(req)

			if tt.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestCreateVMRequest_AllFields(t *testing.T) {
	// Test that all fields can be set and accessed
	req := CreateVMRequest{
		ResourceGroup:      "my-rg",
		VMName:             "my-vm",
		Location:           "westus2",
		Size:               "Standard_D2s_v3",
		AdminUsername:      "adminuser",
		AdminPassword:      "SecureP@ss123",
		NetworkInterfaceID: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/networkInterfaces/nic1",
		ImagePublisher:     "Canonical",
		ImageOffer:         "UbuntuServer",
		ImageSku:           "20.04-LTS",
		ImageVersion:       "latest",
	}

	assert.Equal(t, "my-rg", req.ResourceGroup)
	assert.Equal(t, "my-vm", req.VMName)
	assert.Equal(t, "westus2", req.Location)
	assert.Equal(t, "Standard_D2s_v3", req.Size)
	assert.Equal(t, "adminuser", req.AdminUsername)
	assert.Equal(t, "SecureP@ss123", req.AdminPassword)
	assert.Contains(t, req.NetworkInterfaceID, "networkInterfaces/nic1")
	assert.Equal(t, "Canonical", req.ImagePublisher)
	assert.Equal(t, "UbuntuServer", req.ImageOffer)
	assert.Equal(t, "20.04-LTS", req.ImageSku)
	assert.Equal(t, "latest", req.ImageVersion)
}

func TestCreateVMResponse_AllFields(t *testing.T) {
	// Test that all response fields can be set and accessed
	resp := CreateVMResponse{
		VMID:              "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm1",
		Name:              "vm1",
		ProvisioningState: "Succeeded",
		Location:          "eastus",
		Size:              "Standard_B1s",
		PublicIP:          "20.10.10.10",
		PrivateIP:         "10.0.0.4",
		AdminUsername:     "azureuser",
	}

	assert.Contains(t, resp.VMID, "virtualMachines/vm1")
	assert.Equal(t, "vm1", resp.Name)
	assert.Equal(t, "Succeeded", resp.ProvisioningState)
	assert.Equal(t, "eastus", resp.Location)
	assert.Equal(t, "Standard_B1s", resp.Size)
	assert.Equal(t, "20.10.10.10", resp.PublicIP)
	assert.Equal(t, "10.0.0.4", resp.PrivateIP)
	assert.Equal(t, "azureuser", resp.AdminUsername)
}

func TestVMSizeConstants(t *testing.T) {
	// Verify VM size constants are defined correctly
	assert.Equal(t, "Standard_B1s", VMSizeStandardB1s)
	assert.Equal(t, "Standard_B1ms", VMSizeStandardB1ms)
	assert.Equal(t, "Standard_B2s", VMSizeStandardB2s)
	assert.Equal(t, "Standard_D2s_v3", VMSizeStandardD2sV3)
	assert.Equal(t, "Standard_D4s_v3", VMSizeStandardD4sV3)
	assert.Equal(t, "Standard_D8s_v3", VMSizeStandardD8sV3)
	assert.Equal(t, "Standard_F2s_v2", VMSizeStandardF2sV2)
	assert.Equal(t, "Standard_F4s_v2", VMSizeStandardF4sV2)
	assert.Equal(t, "Standard_F8s_v2", VMSizeStandardF8sV2)
	assert.Equal(t, "Standard_E2s_v3", VMSizeStandardE2sV3)
	assert.Equal(t, "Standard_E4s_v3", VMSizeStandardE4sV3)
	assert.Equal(t, "Standard_E8s_v3", VMSizeStandardE8sV3)
}

func TestImageReferenceConstants(t *testing.T) {
	// Test Ubuntu 18.04 LTS
	assert.Equal(t, "Canonical", ImageUbuntu1804LTS.Publisher)
	assert.Equal(t, "UbuntuServer", ImageUbuntu1804LTS.Offer)
	assert.Equal(t, "18.04-LTS", ImageUbuntu1804LTS.SKU)
	assert.Equal(t, "latest", ImageUbuntu1804LTS.Version)

	// Test Ubuntu 20.04 LTS
	assert.Equal(t, "Canonical", ImageUbuntu2004LTS.Publisher)
	assert.Equal(t, "0001-com-ubuntu-server-focal", ImageUbuntu2004LTS.Offer)
	assert.Equal(t, "20_04-lts-gen2", ImageUbuntu2004LTS.SKU)
	assert.Equal(t, "latest", ImageUbuntu2004LTS.Version)

	// Test Windows Server 2019
	assert.Equal(t, "MicrosoftWindowsServer", ImageWindowsServer2019.Publisher)
	assert.Equal(t, "WindowsServer", ImageWindowsServer2019.Offer)
	assert.Equal(t, "2019-Datacenter", ImageWindowsServer2019.SKU)
	assert.Equal(t, "latest", ImageWindowsServer2019.Version)

	// Test Windows Server 2022
	assert.Equal(t, "MicrosoftWindowsServer", ImageWindowsServer2022.Publisher)
	assert.Equal(t, "WindowsServer", ImageWindowsServer2022.Offer)
	assert.Equal(t, "2022-datacenter-azure-edition", ImageWindowsServer2022.SKU)
	assert.Equal(t, "latest", ImageWindowsServer2022.Version)
}

func TestImageReference_StructFields(t *testing.T) {
	// Test that ImageReference struct works correctly
	img := ImageReference{
		Publisher: "TestPublisher",
		Offer:     "TestOffer",
		SKU:       "TestSKU",
		Version:   "1.0.0",
	}

	assert.Equal(t, "TestPublisher", img.Publisher)
	assert.Equal(t, "TestOffer", img.Offer)
	assert.Equal(t, "TestSKU", img.SKU)
	assert.Equal(t, "1.0.0", img.Version)
}

func TestCreateVMRequest_WithCommonImages(t *testing.T) {
	// Test that CreateVMRequest works with predefined image references
	tests := []struct {
		name  string
		image ImageReference
	}{
		{
			name:  "Ubuntu 18.04 LTS",
			image: ImageUbuntu1804LTS,
		},
		{
			name:  "Ubuntu 20.04 LTS",
			image: ImageUbuntu2004LTS,
		},
		{
			name:  "Windows Server 2019",
			image: ImageWindowsServer2019,
		},
		{
			name:  "Windows Server 2022",
			image: ImageWindowsServer2022,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CreateVMRequest{
				ResourceGroup:      "test-rg",
				VMName:             "test-vm",
				Location:           "eastus",
				Size:               VMSizeStandardB1s,
				AdminUsername:      "azureuser",
				AdminPassword:      "P@ssw0rd123!",
				NetworkInterfaceID: "/subscriptions/test/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic",
				ImagePublisher:     tt.image.Publisher,
				ImageOffer:         tt.image.Offer,
				ImageSku:           tt.image.SKU,
				ImageVersion:       tt.image.Version,
			}

			err := validateCreateVMRequest(req)
			assert.NoError(t, err)
		})
	}
}

func TestCreateVMRequest_WithCommonVMSizes(t *testing.T) {
	// Test that CreateVMRequest works with predefined VM sizes
	vmSizes := []string{
		VMSizeStandardB1s,
		VMSizeStandardB1ms,
		VMSizeStandardB2s,
		VMSizeStandardD2sV3,
		VMSizeStandardD4sV3,
		VMSizeStandardD8sV3,
		VMSizeStandardF2sV2,
		VMSizeStandardF4sV2,
		VMSizeStandardF8sV2,
		VMSizeStandardE2sV3,
		VMSizeStandardE4sV3,
		VMSizeStandardE8sV3,
	}

	for _, size := range vmSizes {
		t.Run(size, func(t *testing.T) {
			req := CreateVMRequest{
				ResourceGroup:      "test-rg",
				VMName:             "test-vm",
				Location:           "eastus",
				Size:               size,
				AdminUsername:      "azureuser",
				AdminPassword:      "P@ssw0rd123!",
				NetworkInterfaceID: "/subscriptions/test/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic",
				ImagePublisher:     "Canonical",
				ImageOffer:         "UbuntuServer",
				ImageSku:           "18.04-LTS",
				ImageVersion:       "latest",
			}

			err := validateCreateVMRequest(req)
			assert.NoError(t, err)
		})
	}
}

// Note: Integration tests for CreateVM would require:
// 1. Valid Azure credentials
// 2. Existing resource group
// 3. Existing network interface
// 4. Permissions to create VMs
// These should be run separately as E2E tests, not unit tests
