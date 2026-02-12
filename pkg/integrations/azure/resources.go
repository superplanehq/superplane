package azure

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeResourceGroupDropdown  = "azure.resourceGroup"
	ResourceTypeVMSizeDropdown         = "azure.vmSize"
	ResourceTypeVirtualNetworkDropdown = "azure.virtualNetwork"
	ResourceTypeSubnetDropdown         = "azure.subnet"
)

func (a *AzureIntegration) ListResourceGroups(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	provider, err := a.providerFromListResourcesContext(ctx)
	if err != nil {
		return nil, err
	}

	pager := provider.GetResourceGroupsClient().NewListPager(nil)
	resources := []core.IntegrationResource{}

	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to list resource groups: %w", err)
		}

		for _, group := range page.Value {
			if group == nil || group.Name == nil {
				continue
			}

			id := *group.Name
			if group.ID != nil {
				id = *group.ID
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeResourceGroupDropdown,
				Name: *group.Name,
				ID:   id,
			})
		}
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

	filter := fmt.Sprintf("location eq '%s'", strings.ToLower(location))
	pager := provider.GetResourceSKUsClient().NewListPager(&armcompute.ResourceSKUsClientListOptions{
		Filter: &filter,
	})
	resourcesByID := map[string]core.IntegrationResource{}

	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to list VM sizes: %w", err)
		}

		for _, sku := range page.Value {
			if !isVirtualMachineSKU(sku) || !isSKUAvailableInLocation(sku, location) || sku.Name == nil {
				continue
			}

			resourcesByID[*sku.Name] = core.IntegrationResource{
				Type: ResourceTypeVMSizeDropdown,
				Name: *sku.Name,
				ID:   *sku.Name,
			}
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

	pager := provider.GetVirtualNetworksClient().NewListPager(resourceGroup, nil)
	resources := []core.IntegrationResource{}

	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to list virtual networks: %w", err)
		}

		for _, vnet := range page.Value {
			if vnet.Name == nil {
				continue
			}

			id := *vnet.Name
			if vnet.ID != nil {
				id = *vnet.ID
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeVirtualNetworkDropdown,
				Name: *vnet.Name,
				ID:   id,
			})
		}
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

	pager := provider.GetSubnetsClient().NewListPager(resourceGroup, vnetName, nil)
	resources := []core.IntegrationResource{}

	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to list subnets: %w", err)
		}

		for _, subnet := range page.Value {
			if subnet.Name == nil {
				continue
			}

			id := *subnet.Name
			if subnet.ID != nil {
				id = *subnet.ID
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeSubnetDropdown,
				Name: *subnet.Name,
				ID:   id,
			})
		}
	}

	sort.Slice(resources, func(i, j int) bool {
		return strings.ToLower(resources[i].Name) < strings.ToLower(resources[j].Name)
	})

	return resources, nil
}

func (a *AzureIntegration) providerFromListResourcesContext(ctx core.ListResourcesContext) (*AzureProvider, error) {
	if a.provider != nil {
		return a.provider, nil
	}

	if ctx.Integration == nil {
		return nil, fmt.Errorf("integration context is required")
	}

	tenantID, err := ctx.Integration.GetConfig("tenantId")
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant ID: %w", err)
	}

	clientID, err := ctx.Integration.GetConfig("clientId")
	if err != nil {
		return nil, fmt.Errorf("failed to get client ID: %w", err)
	}

	subscriptionID, err := ctx.Integration.GetConfig("subscriptionId")
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription ID: %w", err)
	}

	provider, err := NewAzureProvider(
		context.Background(),
		string(tenantID),
		string(clientID),
		string(subscriptionID),
		ctx.Logger,
	)
	if err != nil {
		return nil, err
	}

	a.provider = provider
	return provider, nil
}

func isVirtualMachineSKU(sku *armcompute.ResourceSKU) bool {
	if sku == nil || sku.ResourceType == nil {
		return false
	}

	return strings.EqualFold(*sku.ResourceType, "virtualMachines")
}

func isSKUAvailableInLocation(sku *armcompute.ResourceSKU, location string) bool {
	if sku == nil {
		return false
	}

	for _, skuLocation := range sku.Locations {
		if skuLocation != nil && strings.EqualFold(*skuLocation, location) {
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
	// (e.g. %2Fsubscriptions%2F... instead of /subscriptions/...).
	if decoded, err := url.QueryUnescape(trimmed); err == nil {
		trimmed = decoded
	}

	if !strings.Contains(trimmed, "/") {
		return trimmed
	}

	return path.Base(strings.TrimRight(trimmed, "/"))
}
