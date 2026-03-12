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

func TestImageReferenceConstants(t *testing.T) {
	assert.Equal(t, "Canonical", ImageUbuntu2004LTS.Publisher)
	assert.Equal(t, "0001-com-ubuntu-server-focal", ImageUbuntu2004LTS.Offer)
	assert.Equal(t, "20_04-lts-gen2", ImageUbuntu2004LTS.SKU)
	assert.Equal(t, "latest", ImageUbuntu2004LTS.Version)

	assert.Equal(t, "Canonical", ImageUbuntu2204LTS.Publisher)
	assert.Equal(t, "Canonical", ImageUbuntu2404LTS.Publisher)
	assert.Equal(t, "Debian", ImageDebian12.Publisher)
	assert.Equal(t, "MicrosoftWindowsServer", ImageWindowsServer2022.Publisher)
}

func TestResolveImagePreset(t *testing.T) {
	img := ResolveImagePreset("ubuntu-22.04")
	assert.Equal(t, ImageUbuntu2204LTS, img)

	img = ResolveImagePreset("windows-2022")
	assert.Equal(t, ImageWindowsServer2022, img)

	// Unknown preset falls back to Ubuntu 24.04
	img = ResolveImagePreset("unknown")
	assert.Equal(t, ImageUbuntu2404LTS, img)

	// Empty preset falls back to Ubuntu 24.04
	img = ResolveImagePreset("")
	assert.Equal(t, ImageUbuntu2404LTS, img)
}

func TestImageReference_StructFields(t *testing.T) {
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

func TestValidateDeleteVMRequest(t *testing.T) {
	validRequest := DeleteVMRequest{
		ResourceGroup: "test-rg",
		VMName:        "test-vm",
	}

	tests := []struct {
		name        string
		modifyReq   func(*DeleteVMRequest)
		expectedErr string
	}{
		{
			name:        "valid request",
			modifyReq:   func(r *DeleteVMRequest) {},
			expectedErr: "",
		},
		{
			name: "missing resource group",
			modifyReq: func(r *DeleteVMRequest) {
				r.ResourceGroup = ""
			},
			expectedErr: "resource group is required",
		},
		{
			name: "missing VM name",
			modifyReq: func(r *DeleteVMRequest) {
				r.VMName = ""
			},
			expectedErr: "VM name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest
			tt.modifyReq(&req)

			err := validateDeleteVMRequest(req)

			if tt.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestDeleteVMRequest_AllFields(t *testing.T) {
	req := DeleteVMRequest{
		ResourceGroup: "my-rg",
		VMName:        "my-vm",
	}

	assert.Equal(t, "my-rg", req.ResourceGroup)
	assert.Equal(t, "my-vm", req.VMName)
}

func TestCreateVMRequest_WithUbuntuImage(t *testing.T) {
	req := CreateVMRequest{
		ResourceGroup:      "test-rg",
		VMName:             "test-vm",
		Location:           "eastus",
		Size:               "Standard_B1s",
		AdminUsername:      "azureuser",
		AdminPassword:      "P@ssw0rd123!",
		NetworkInterfaceID: "/subscriptions/test/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic",
		ImagePublisher:     ImageUbuntu2004LTS.Publisher,
		ImageOffer:         ImageUbuntu2004LTS.Offer,
		ImageSku:           ImageUbuntu2004LTS.SKU,
		ImageVersion:       ImageUbuntu2004LTS.Version,
	}

	err := validateCreateVMRequest(req)
	assert.NoError(t, err)
}
