package compute

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/azure/common"
)

const (
	computeAPIVersion = "2024-07-01"
	armBaseURL        = "https://management.azure.com"
)

type Client struct {
	http            core.HTTPContext
	credentials     *common.Credentials
	subscriptionID  string
}

func NewClient(httpCtx core.HTTPContext, credentials *common.Credentials, subscriptionID string) *Client {
	return &Client{
		http:           httpCtx,
		credentials:    credentials,
		subscriptionID: strings.TrimSpace(subscriptionID),
	}
}

// CreateOrUpdateVMParams holds the ARM request body for VM create/update.
type CreateOrUpdateVMParams struct {
	Location   string                 `json:"location"`
	Properties CreateOrUpdateVMProps  `json:"properties"`
	Tags       map[string]string      `json:"tags,omitempty"`
}

type CreateOrUpdateVMProps struct {
	HardwareProfile HardwareProfile `json:"hardwareProfile"`
	StorageProfile  StorageProfile `json:"storageProfile"`
	OSProfile       OSProfile      `json:"osProfile"`
	NetworkProfile  NetworkProfile `json:"networkProfile"`
}

type HardwareProfile struct {
	VMSize string `json:"vmSize"`
}

type StorageProfile struct {
	ImageReference ImageReference `json:"imageReference"`
	OSDisk         OSDisk         `json:"osDisk"`
}

type ImageReference struct {
	Publisher string `json:"publisher"`
	Offer     string `json:"offer"`
	SKU       string `json:"sku"`
	Version   string `json:"version,omitempty"`
}

// ManagedDiskOptions is used in OSDisk.
type ManagedDiskOptions struct {
	StorageAccountType string `json:"storageAccountType,omitempty"`
}

type OSDisk struct {
	CreateOption string              `json:"createOption"`
	Caching      string              `json:"caching,omitempty"`
	ManagedDisk  *ManagedDiskOptions `json:"managedDisk,omitempty"`
	DiskSizeGB   int                 `json:"diskSizeGB,omitempty"`
}

type OSProfile struct {
	ComputerName       string            `json:"computerName"`
	AdminUsername      string            `json:"adminUsername"`
	AdminPassword      string            `json:"adminPassword,omitempty"`
	LinuxConfiguration *LinuxConfiguration `json:"linuxConfiguration,omitempty"`
	WindowsConfiguration *WindowsConfiguration `json:"windowsConfiguration,omitempty"` //nolint:revive // struct field name matches type
}

type LinuxConfiguration struct {
	DisablePasswordAuthentication bool        `json:"disablePasswordAuthentication,omitempty"`
	SSH                           *SSHConfig  `json:"ssh,omitempty"`
}

type SSHConfig struct {
	PublicKeys []SSHPublicKey `json:"publicKeys"`
}

type SSHPublicKey struct {
	Path    string `json:"path"`
	KeyData string `json:"keyData"`
}

type WindowsConfiguration struct {
	ProvisionVMAgent bool `json:"provisionVMAgent,omitempty"`
}

type NetworkProfile struct {
	NetworkInterfaces []NetworkInterfaceRef `json:"networkInterfaces"`
}

type NetworkInterfaceRef struct {
	ID string `json:"id"`
}

// VMModel is the ARM response for a VM (subset we use).
type VMModel struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Location   string             `json:"location"`
	Type       string             `json:"type"`
	Properties *VMModelProperties  `json:"properties,omitempty"`
}

type VMModelProperties struct {
	ProvisioningState string `json:"provisioningState,omitempty"`
	VMID              string `json:"vmId,omitempty"`
	HardwareProfile   *struct {
		VMSize string `json:"vmSize,omitempty"`
	} `json:"hardwareProfile,omitempty"`
}

// CreateOrUpdateVM creates or updates a virtual machine.
func (c *Client) CreateOrUpdateVM(resourceGroup, name string, params *CreateOrUpdateVMParams) (*VMModel, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s",
		c.subscriptionID, resourceGroup, name)
	url := armBaseURL + path + "?api-version=" + computeAPIVersion

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal VM body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.credentials.AccessToken)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("VM create request failed: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		if azErr := common.ParseARMError(resBody); azErr != nil {
			return nil, azErr
		}
		return nil, fmt.Errorf("VM create failed with %d: %s", res.StatusCode, string(resBody))
	}

	var vm VMModel
	if err := json.Unmarshal(resBody, &vm); err != nil {
		return nil, fmt.Errorf("failed to decode VM response: %w", err)
	}
	return &vm, nil
}

// ListVMsResult holds one VM summary for listing.
type ListVMsResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

// ListVMs lists virtual machines in a resource group.
func (c *Client) ListVMs(resourceGroup string) ([]ListVMsResult, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines",
		c.subscriptionID, resourceGroup)
	url := armBaseURL + path + "?api-version=" + computeAPIVersion

	var all []ListVMsResult
	for {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build list request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.credentials.AccessToken)

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list VMs request failed: %w", err)
		}

		resBody, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read list response: %w", err)
		}

		if res.StatusCode != http.StatusOK {
			if azErr := common.ParseARMError(resBody); azErr != nil {
				return nil, azErr
			}
			return nil, fmt.Errorf("list VMs failed with %d: %s", res.StatusCode, string(resBody))
		}

		var page struct {
			Value    []ListVMsResult `json:"value"`
			NextLink string          `json:"nextLink"`
		}
		if err := json.Unmarshal(resBody, &page); err != nil {
			return nil, fmt.Errorf("failed to decode list response: %w", err)
		}
		all = append(all, page.Value...)
		if strings.TrimSpace(page.NextLink) == "" {
			break
		}
		url = page.NextLink
	}
	return all, nil
}

// ListVMsBySubscription lists all virtual machines in the subscription (all resource groups).
func (c *Client) ListVMsBySubscription() ([]ListVMsResult, error) {
	path := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/virtualMachines", c.subscriptionID)
	url := armBaseURL + path + "?api-version=" + computeAPIVersion

	var all []ListVMsResult
	for {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build list request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.credentials.AccessToken)

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list VMs request failed: %w", err)
		}

		resBody, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read list response: %w", err)
		}

		if res.StatusCode != http.StatusOK {
			if azErr := common.ParseARMError(resBody); azErr != nil {
				return nil, azErr
			}
			return nil, fmt.Errorf("list VMs failed with %d: %s", res.StatusCode, string(resBody))
		}

		var page struct {
			Value    []ListVMsResult `json:"value"`
			NextLink string          `json:"nextLink"`
		}
		if err := json.Unmarshal(resBody, &page); err != nil {
			return nil, fmt.Errorf("failed to decode list response: %w", err)
		}
		all = append(all, page.Value...)
		if strings.TrimSpace(page.NextLink) == "" {
			break
		}
		url = page.NextLink
	}
	return all, nil
}
