package azure

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func newSetupContext(config map[string]any) core.SetupContext {
	return core.SetupContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: config,
		Metadata:      &contexts.MetadataContext{},
	}
}

func TestDeleteVMComponent_Metadata(t *testing.T) {
	component := &DeleteVMComponent{}

	assert.Equal(t, "azure.deleteVirtualMachine", component.Name())
	assert.Equal(t, "Delete Virtual Machine", component.Label())
	assert.Equal(t, "azure", component.Icon())
	assert.Equal(t, "blue", component.Color())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
	assert.Contains(t, component.Description(), "Deletes")
}

func TestDeleteVMComponent_Configuration(t *testing.T) {
	component := &DeleteVMComponent{}
	fields := component.Configuration()

	require.Len(t, fields, 3, "Should have exactly 3 configuration fields")

	resourceGroupField := fields[0]
	assert.Equal(t, "resourceGroup", resourceGroupField.Name)
	assert.Equal(t, "Resource Group", resourceGroupField.Label)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, resourceGroupField.Type)
	assert.True(t, resourceGroupField.Required)
	assert.NotEmpty(t, resourceGroupField.Description)

	nameField := fields[1]
	assert.Equal(t, "name", nameField.Name)
	assert.Equal(t, "VM Name", nameField.Label)
	assert.Equal(t, configuration.FieldTypeString, nameField.Type)
	assert.True(t, nameField.Required)
	assert.NotEmpty(t, nameField.Description)

	deleteAssocField := fields[2]
	assert.Equal(t, "deleteAssociatedResources", deleteAssocField.Name)
	assert.Equal(t, "Delete Associated Resources", deleteAssocField.Label)
	assert.Equal(t, configuration.FieldTypeBool, deleteAssocField.Type)
	assert.False(t, deleteAssocField.Required)
	assert.NotEmpty(t, deleteAssocField.Description)
}

func TestDeleteVMComponent_ExampleOutput(t *testing.T) {
	component := &DeleteVMComponent{}
	example := component.ExampleOutput()

	require.NotNil(t, example)
	assert.Contains(t, example, "data")
	assert.Contains(t, example, "timestamp")
	assert.Contains(t, example, "type")

	data, ok := example["data"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, data, "id")
	assert.Contains(t, data, "name")
	assert.Contains(t, data, "resourceGroup")
	assert.Equal(t, "my-vm", data["name"])
	assert.Equal(t, "my-rg", data["resourceGroup"])
}

func TestDeleteVMComponent_OutputChannels(t *testing.T) {
	component := &DeleteVMComponent{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestDeleteVMComponent_Actions(t *testing.T) {
	component := &DeleteVMComponent{}
	actions := component.Actions()

	assert.NotNil(t, actions)
	assert.Empty(t, actions)
}

func TestDeleteVMComponent_HandleAction(t *testing.T) {
	component := &DeleteVMComponent{}

	ctx := core.ActionContext{
		Name:   "test",
		Logger: logrus.NewEntry(logrus.New()),
	}

	err := component.HandleAction(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions defined")
}

func TestDeleteVMComponent_Setup_Valid(t *testing.T) {
	component := &DeleteVMComponent{}

	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
		"name":          "my-vm",
	})

	err := component.Setup(ctx)
	assert.NoError(t, err)
}

func TestDeleteVMComponent_Setup_ValidWithDeleteAssociated(t *testing.T) {
	component := &DeleteVMComponent{}

	ctx := newSetupContext(map[string]any{
		"resourceGroup":             "my-rg",
		"name":                      "my-vm",
		"deleteAssociatedResources": true,
	})

	err := component.Setup(ctx)
	assert.NoError(t, err)
}

func TestDeleteVMComponent_Setup_MissingResourceGroup(t *testing.T) {
	component := &DeleteVMComponent{}

	ctx := newSetupContext(map[string]any{
		"name": "my-vm",
	})

	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource group is required")
}

func TestDeleteVMComponent_Setup_MissingName(t *testing.T) {
	component := &DeleteVMComponent{}

	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
	})

	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VM name is required")
}

func TestDeleteVM_WithoutAssociatedResources(t *testing.T) {
	_, server := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "virtualMachines/test-vm") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"status": "Succeeded"})
			return
		}

		// No GET or PATCH should be called when deleteAssociatedResources is false.
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	client := &armClient{
		subscriptionID: "test-sub",
		baseURL:        server.URL,
		tokenFunc:      func(_ context.Context) (string, error) { return "mock-token", nil },
		httpClient:     &http.Client{},
		logger:         logrus.NewEntry(logrus.New()),
	}
	provider := &AzureProvider{
		subscriptionID: "test-sub",
		client:         client,
		logger:         logrus.NewEntry(logrus.New()),
	}

	req := DeleteVMRequest{
		ResourceGroup:             "test-rg",
		VMName:                    "test-vm",
		DeleteAssociatedResources: false,
	}

	resp, err := DeleteVM(context.Background(), provider, req, logrus.NewEntry(logrus.New()))
	require.NoError(t, err)
	assert.Equal(t, "test-vm", resp["name"])
	assert.Equal(t, "test-rg", resp["resourceGroup"])
}

func TestDeleteVM_WithAssociatedResources(t *testing.T) {
	var mu sync.Mutex
	requestLog := []string{}

	vmResponse := armVM{
		ID:       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm",
		Name:     "test-vm",
		Location: "eastus",
		Properties: &armVMProperties{
			StorageProfile: &armStorageProfile{
				OsDisk: &armOsDisk{
					ManagedDisk: &armManagedDiskRef{
						ID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/disks/test-vm-osdisk",
					},
				},
				DataDisks: []armDataDisk{
					{
						Lun: 0,
						ManagedDisk: &armManagedDiskRef{
							ID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/disks/test-vm-datadisk",
						},
					},
				},
			},
			NetworkProfile: &armNetworkProfile{
				NetworkInterfaces: []armNetworkInterfaceRef{
					{ID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic"},
				},
			},
		},
	}

	nicResponse := armNIC{
		ID:       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic",
		Location: "eastus",
		Properties: &armNICProperties{
			IPConfigurations: []armIPConfiguration{
				{
					Name: "ipconfig1",
					Properties: &armIPConfigProperties{
						PrivateIPAddress: "10.0.0.4",
						PublicIPAddress: &armPublicIPRef{
							ID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/publicIPAddresses/test-pip",
						},
						Subnet: &armSubnetResponse{
							ID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/default",
						},
					},
				},
			},
		},
	}

	_, server := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestLog = append(requestLog, r.Method+" "+r.URL.Path)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		// GET VM
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "virtualMachines/test-vm") {
			json.NewEncoder(w).Encode(vmResponse)
			return
		}

		// PATCH VM (set deleteOption on disks and NICs)
		if r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "virtualMachines/test-vm") {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)

			props, _ := body["properties"].(map[string]any)
			require.NotNil(t, props)

			// Verify storage profile has deleteOption
			sp, _ := props["storageProfile"].(map[string]any)
			require.NotNil(t, sp)
			osDisk, _ := sp["osDisk"].(map[string]any)
			assert.Equal(t, "Delete", osDisk["deleteOption"])

			// Verify network profile has deleteOption
			np, _ := props["networkProfile"].(map[string]any)
			require.NotNil(t, np)

			json.NewEncoder(w).Encode(vmResponse)
			return
		}

		// GET NIC
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "networkInterfaces/test-nic") {
			json.NewEncoder(w).Encode(nicResponse)
			return
		}

		// PATCH NIC (set deleteOption on public IP)
		if r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "networkInterfaces/test-nic") {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)

			props, _ := body["properties"].(map[string]any)
			require.NotNil(t, props)

			ipConfigs, _ := props["ipConfigurations"].([]any)
			require.Len(t, ipConfigs, 1)

			json.NewEncoder(w).Encode(nicResponse)
			return
		}

		// DELETE VM
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "virtualMachines/test-vm") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"status": "Succeeded"})
			return
		}

		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	client := &armClient{
		subscriptionID: "test-sub",
		baseURL:        server.URL,
		tokenFunc:      func(_ context.Context) (string, error) { return "mock-token", nil },
		httpClient:     &http.Client{},
		logger:         logrus.NewEntry(logrus.New()),
	}
	provider := &AzureProvider{
		subscriptionID: "test-sub",
		client:         client,
		logger:         logrus.NewEntry(logrus.New()),
	}

	req := DeleteVMRequest{
		ResourceGroup:             "test-rg",
		VMName:                    "test-vm",
		DeleteAssociatedResources: true,
	}

	resp, err := DeleteVM(context.Background(), provider, req, logrus.NewEntry(logrus.New()))
	require.NoError(t, err)
	assert.Equal(t, "test-vm", resp["name"])
	assert.Equal(t, "test-rg", resp["resourceGroup"])

	// Verify the request sequence: GET VM, PATCH VM, GET NIC, PATCH NIC, DELETE VM
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, requestLog, 5)
	assert.Contains(t, requestLog[0], "GET")
	assert.Contains(t, requestLog[0], "virtualMachines")
	assert.Contains(t, requestLog[1], "PATCH")
	assert.Contains(t, requestLog[1], "virtualMachines")
	assert.Contains(t, requestLog[2], "GET")
	assert.Contains(t, requestLog[2], "networkInterfaces")
	assert.Contains(t, requestLog[3], "PATCH")
	assert.Contains(t, requestLog[3], "networkInterfaces")
	assert.Contains(t, requestLog[4], "DELETE")
	assert.Contains(t, requestLog[4], "virtualMachines")
}

func TestDeleteVM_WithAssociatedResources_NoPublicIP(t *testing.T) {
	var mu sync.Mutex
	requestLog := []string{}

	vmResponse := armVM{
		ID:       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm",
		Name:     "test-vm",
		Location: "eastus",
		Properties: &armVMProperties{
			StorageProfile: &armStorageProfile{
				OsDisk: &armOsDisk{
					ManagedDisk: &armManagedDiskRef{
						ID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/disks/test-vm-osdisk",
					},
				},
			},
			NetworkProfile: &armNetworkProfile{
				NetworkInterfaces: []armNetworkInterfaceRef{
					{ID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic"},
				},
			},
		},
	}

	nicResponse := armNIC{
		ID:       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic",
		Location: "eastus",
		Properties: &armNICProperties{
			IPConfigurations: []armIPConfiguration{
				{
					Name: "ipconfig1",
					Properties: &armIPConfigProperties{
						PrivateIPAddress: "10.0.0.4",
						Subnet: &armSubnetResponse{
							ID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/default",
						},
					},
				},
			},
		},
	}

	_, server := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestLog = append(requestLog, r.Method+" "+r.URL.Path)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "virtualMachines/test-vm") {
			json.NewEncoder(w).Encode(vmResponse)
			return
		}
		if r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "virtualMachines/test-vm") {
			json.NewEncoder(w).Encode(vmResponse)
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "networkInterfaces/test-nic") {
			json.NewEncoder(w).Encode(nicResponse)
			return
		}
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "virtualMachines/test-vm") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"status": "Succeeded"})
			return
		}

		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	client := &armClient{
		subscriptionID: "test-sub",
		baseURL:        server.URL,
		tokenFunc:      func(_ context.Context) (string, error) { return "mock-token", nil },
		httpClient:     &http.Client{},
		logger:         logrus.NewEntry(logrus.New()),
	}
	provider := &AzureProvider{
		subscriptionID: "test-sub",
		client:         client,
		logger:         logrus.NewEntry(logrus.New()),
	}

	req := DeleteVMRequest{
		ResourceGroup:             "test-rg",
		VMName:                    "test-vm",
		DeleteAssociatedResources: true,
	}

	resp, err := DeleteVM(context.Background(), provider, req, logrus.NewEntry(logrus.New()))
	require.NoError(t, err)
	assert.Equal(t, "test-vm", resp["name"])

	// Should NOT PATCH the NIC since there's no public IP.
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, requestLog, 4, "Should be GET VM, PATCH VM, GET NIC, DELETE VM (no PATCH NIC)")
	assert.Contains(t, requestLog[0], "GET")
	assert.Contains(t, requestLog[1], "PATCH")
	assert.Contains(t, requestLog[1], "virtualMachines")
	assert.Contains(t, requestLog[2], "GET")
	assert.Contains(t, requestLog[2], "networkInterfaces")
	assert.Contains(t, requestLog[3], "DELETE")
}

func TestBuildVMDeleteOptionPatch(t *testing.T) {
	t.Run("nil properties returns nil", func(t *testing.T) {
		vm := &armVM{}
		patch := buildVMDeleteOptionPatch(vm)
		assert.Nil(t, patch)
	})

	t.Run("with OS disk and data disks", func(t *testing.T) {
		vm := &armVM{
			Properties: &armVMProperties{
				StorageProfile: &armStorageProfile{
					OsDisk: &armOsDisk{
						ManagedDisk: &armManagedDiskRef{ID: "os-disk-id"},
					},
					DataDisks: []armDataDisk{
						{Lun: 0, ManagedDisk: &armManagedDiskRef{ID: "data-disk-0"}},
						{Lun: 1, ManagedDisk: &armManagedDiskRef{ID: "data-disk-1"}},
					},
				},
				NetworkProfile: &armNetworkProfile{
					NetworkInterfaces: []armNetworkInterfaceRef{
						{ID: "nic-1"},
						{ID: "nic-2"},
					},
				},
			},
		}

		patch := buildVMDeleteOptionPatch(vm)
		require.NotNil(t, patch)

		props := patch["properties"].(map[string]any)

		// Check storage profile
		sp := props["storageProfile"].(map[string]any)
		osDisk := sp["osDisk"].(map[string]any)
		assert.Equal(t, "Delete", osDisk["deleteOption"])

		dataDisks := sp["dataDisks"].([]map[string]any)
		require.Len(t, dataDisks, 2)
		assert.Equal(t, 0, dataDisks[0]["lun"])
		assert.Equal(t, "Delete", dataDisks[0]["deleteOption"])
		assert.Equal(t, 1, dataDisks[1]["lun"])
		assert.Equal(t, "Delete", dataDisks[1]["deleteOption"])

		// Check network profile
		np := props["networkProfile"].(map[string]any)
		nics := np["networkInterfaces"].([]map[string]any)
		require.Len(t, nics, 2)
		assert.Equal(t, "nic-1", nics[0]["id"])
		nicProps := nics[0]["properties"].(map[string]any)
		assert.Equal(t, "Delete", nicProps["deleteOption"])
	})

	t.Run("no storage profile or network profile", func(t *testing.T) {
		vm := &armVM{
			Properties: &armVMProperties{},
		}
		patch := buildVMDeleteOptionPatch(vm)
		assert.Nil(t, patch)
	})
}
