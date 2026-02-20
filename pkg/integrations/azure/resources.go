package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeResourceGroupDropdown  = "azure.resourceGroup"
	ResourceTypeVMSizeDropdown         = "azure.vmSize"
	ResourceTypeVirtualNetworkDropdown = "azure.virtualNetwork"
	ResourceTypeSubnetDropdown         = "azure.subnet"
)

type armResourceGroup struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type armSKU struct {
	Name         string   `json:"name"`
	ResourceType string   `json:"resourceType"`
	Locations    []string `json:"locations"`
}

type armVirtualNetwork struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type armSubnet struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func (a *AzureIntegration) ListResourceGroups(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	provider, err := a.providerFromListResourcesContext(ctx)
	if err != nil {
		return nil, err
	}

	listURL := fmt.Sprintf("%s/subscriptions/%s/resourcegroups?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), armAPIVersionResources)

	items, err := provider.GetClient().listAll(context.Background(), listURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource groups: %w", err)
	}

	resources := []core.IntegrationResource{}
	for _, raw := range items {
		var group armResourceGroup
		if err := json.Unmarshal(raw, &group); err != nil {
			continue
		}
		if group.Name == "" {
			continue
		}

		id := group.Name
		if group.ID != "" {
			id = group.ID
		}

		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeResourceGroupDropdown,
			Name: group.Name,
			ID:   id,
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return strings.ToLower(resources[i].Name) < strings.ToLower(resources[j].Name)
	})

	return resources, nil
}

func (a *AzureIntegration) ListVMSizes(ctx core.ListResourcesContext, location string) ([]core.IntegrationResource, error) {
	if location == "" {
		return []core.IntegrationResource{}, nil
	}

	provider, err := a.providerFromListResourcesContext(ctx)
	if err != nil {
		return nil, err
	}

	listURL := fmt.Sprintf("%s/subscriptions/%s/providers/Microsoft.Compute/skus?api-version=%s&$filter=location eq '%s'",
		armBaseURL, provider.GetSubscriptionID(), armAPIVersionCompute, strings.ToLower(location))

	items, err := provider.GetClient().listAll(context.Background(), listURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list VM sizes: %w", err)
	}

	resourcesByID := map[string]core.IntegrationResource{}
	for _, raw := range items {
		var sku armSKU
		if err := json.Unmarshal(raw, &sku); err != nil {
			continue
		}

		if !isVirtualMachineSKU(sku) || !isSKUAvailableInLocation(sku, location) || sku.Name == "" {
			continue
		}

		resourcesByID[sku.Name] = core.IntegrationResource{
			Type: ResourceTypeVMSizeDropdown,
			Name: sku.Name,
			ID:   sku.Name,
		}
	}

	resources := make([]core.IntegrationResource, 0, len(resourcesByID))
	for _, item := range resourcesByID {
		resources = append(resources, item)
	}

	sort.Slice(resources, func(i, j int) bool {
		return strings.ToLower(resources[i].Name) < strings.ToLower(resources[j].Name)
	})

	return resources, nil
}

func (a *AzureIntegration) ListVirtualNetworks(ctx core.ListResourcesContext, resourceGroup string) ([]core.IntegrationResource, error) {
	if resourceGroup == "" {
		return []core.IntegrationResource{}, nil
	}
	resourceGroup = azureResourceName(resourceGroup)

	provider, err := a.providerFromListResourcesContext(ctx)
	if err != nil {
		return nil, err
	}

	listURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), resourceGroup, armAPIVersionNetwork)

	items, err := provider.GetClient().listAll(context.Background(), listURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual networks: %w", err)
	}

	resources := []core.IntegrationResource{}
	for _, raw := range items {
		var vnet armVirtualNetwork
		if err := json.Unmarshal(raw, &vnet); err != nil {
			continue
		}
		if vnet.Name == "" {
			continue
		}

		id := vnet.Name
		if vnet.ID != "" {
			id = vnet.ID
		}

		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeVirtualNetworkDropdown,
			Name: vnet.Name,
			ID:   id,
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return strings.ToLower(resources[i].Name) < strings.ToLower(resources[j].Name)
	})

	return resources, nil
}

func (a *AzureIntegration) ListSubnets(ctx core.ListResourcesContext, resourceGroup, vnetName string) ([]core.IntegrationResource, error) {
	if resourceGroup == "" || vnetName == "" {
		return []core.IntegrationResource{}, nil
	}
	resourceGroup = azureResourceName(resourceGroup)
	vnetName = azureResourceName(vnetName)

	provider, err := a.providerFromListResourcesContext(ctx)
	if err != nil {
		return nil, err
	}

	listURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), resourceGroup, vnetName, armAPIVersionNetwork)

	items, err := provider.GetClient().listAll(context.Background(), listURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list subnets: %w", err)
	}

	resources := []core.IntegrationResource{}
	for _, raw := range items {
		var subnet armSubnet
		if err := json.Unmarshal(raw, &subnet); err != nil {
			continue
		}
		if subnet.Name == "" {
			continue
		}

		id := subnet.Name
		if subnet.ID != "" {
			id = subnet.ID
		}

		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeSubnetDropdown,
			Name: subnet.Name,
			ID:   id,
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return strings.ToLower(resources[i].Name) < strings.ToLower(resources[j].Name)
	})

	return resources, nil
}

func (a *AzureIntegration) providerFromListResourcesContext(_ core.ListResourcesContext) (*AzureProvider, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("Azure provider not initialized; Sync must run before listing resources")
	}

	return a.provider, nil
}

func isVirtualMachineSKU(sku armSKU) bool {
	return strings.EqualFold(sku.ResourceType, "virtualMachines")
}

func isSKUAvailableInLocation(sku armSKU, location string) bool {
	for _, skuLocation := range sku.Locations {
		if strings.EqualFold(skuLocation, location) {
			return true
		}
	}
	return false
}

// azureResourceName extracts the final resource name segment from an Azure resource ID.
// It handles plain names, full ARM IDs, and URL-encoded ARM IDs (e.g. from query parameters).
func azureResourceName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	// URL-decode first to handle values that arrive encoded from the query parser
	if decoded, err := url.QueryUnescape(trimmed); err == nil {
		trimmed = decoded
	}

	if !strings.Contains(trimmed, "/") {
		return trimmed
	}

	return path.Base(strings.TrimRight(trimmed, "/"))
}
