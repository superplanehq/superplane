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
	ResourceTypeResourceGroupDropdown     = "azure.resourceGroup"
	ResourceTypeResourceGroupLocation     = "azure.resourceGroupLocation"
	ResourceTypeVMSizeDropdown            = "azure.vmSize"
	ResourceTypeVirtualNetworkDropdown    = "azure.virtualNetwork"
	ResourceTypeSubnetDropdown            = "azure.subnet"
	ResourceTypeContainerRegistryDropdown = "azure.containerRegistry"
)

type armResourceGroup struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Location string `json:"location"`
}

type armVMSize struct {
	Name string `json:"name"`
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
	ctx.Logger.Info("listing Azure resource groups")
	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return nil, err
	}
	return listResourceGroups(ctx, provider)
}

func listResourceGroups(ctx core.ListResourcesContext, provider *AzureProvider) ([]core.IntegrationResource, error) {
	listURL := fmt.Sprintf("%s/subscriptions/%s/resourcegroups?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), armAPIVersionResources)

	items, err := provider.getClient().listAll(context.Background(), listURL)
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

func getResourceGroupLocation(ctx context.Context, provider *AzureProvider, resourceGroup string) (string, error) {
	getURL := fmt.Sprintf("%s/subscriptions/%s/resourcegroups/%s?api-version=%s",
		provider.getClient().getBaseURL(), provider.GetSubscriptionID(), resourceGroup, armAPIVersionResources)

	var group armResourceGroup
	if err := provider.getClient().get(ctx, getURL, &group); err != nil {
		return "", fmt.Errorf("failed to get resource group: %w", err)
	}

	return group.Location, nil
}

func (a *AzureIntegration) ListResourceGroupLocations(ctx core.ListResourcesContext, resourceGroup string) ([]core.IntegrationResource, error) {
	if resourceGroup == "" {
		return []core.IntegrationResource{}, nil
	}
	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return nil, err
	}
	return listResourceGroupLocations(ctx, provider, resourceGroup)
}

func listResourceGroupLocations(ctx core.ListResourcesContext, provider *AzureProvider, resourceGroup string) ([]core.IntegrationResource, error) {
	resourceGroup = azureResourceName(resourceGroup)

	location, err := getResourceGroupLocation(context.Background(), provider, resourceGroup)
	if err != nil {
		if isARMNotFound(err) {
			ctx.Logger.Warnf("resource group %s not found, returning empty location", resourceGroup)
			return []core.IntegrationResource{}, nil
		}
		return nil, err
	}

	if location == "" {
		return []core.IntegrationResource{}, nil
	}

	return []core.IntegrationResource{
		{
			Type: ResourceTypeResourceGroupLocation,
			Name: location,
			ID:   location,
		},
	}, nil
}

func (a *AzureIntegration) ListVMSizes(ctx core.ListResourcesContext, resourceGroup string) ([]core.IntegrationResource, error) {
	ctx.Logger.Infof("listing Azure VM sizes for resourceGroup=%s", resourceGroup)
	if resourceGroup == "" {
		return []core.IntegrationResource{}, nil
	}
	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return nil, err
	}
	return listVMSizes(ctx, provider, resourceGroup)
}

func listVMSizes(ctx core.ListResourcesContext, provider *AzureProvider, resourceGroup string) ([]core.IntegrationResource, error) {
	resourceGroup = azureResourceName(resourceGroup)

	location, err := getResourceGroupLocation(context.Background(), provider, resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve location for resource group: %w", err)
	}

	if location == "" {
		return []core.IntegrationResource{}, nil
	}

	ctx.Logger.Infof("resolved location=%s for resourceGroup=%s", location, resourceGroup)

	listURL := fmt.Sprintf("%s/subscriptions/%s/providers/Microsoft.Compute/locations/%s/vmSizes?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), strings.ToLower(location), armAPIVersionCompute)

	items, err := provider.getClient().listAll(context.Background(), listURL)
	if err != nil {
		ctx.Logger.WithError(err).Errorf("failed to list VM sizes for location=%s", location)
		return nil, fmt.Errorf("failed to list VM sizes: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(items))
	for _, raw := range items {
		var vmSize armVMSize
		if err := json.Unmarshal(raw, &vmSize); err != nil || vmSize.Name == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeVMSizeDropdown,
			Name: vmSize.Name,
			ID:   vmSize.Name,
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return strings.ToLower(resources[i].Name) < strings.ToLower(resources[j].Name)
	})

	return resources, nil
}

func (a *AzureIntegration) ListVirtualNetworks(ctx core.ListResourcesContext, resourceGroup string) ([]core.IntegrationResource, error) {
	ctx.Logger.Infof("listing Azure virtual networks for resourceGroup=%s", resourceGroup)
	if resourceGroup == "" {
		return []core.IntegrationResource{}, nil
	}
	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return nil, err
	}
	return listVirtualNetworks(ctx, provider, resourceGroup)
}

func listVirtualNetworks(ctx core.ListResourcesContext, provider *AzureProvider, resourceGroup string) ([]core.IntegrationResource, error) {
	resourceGroup = azureResourceName(resourceGroup)

	listURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), resourceGroup, armAPIVersionNetwork)

	items, err := provider.getClient().listAll(context.Background(), listURL)
	if err != nil {
		if isARMNotFound(err) {
			ctx.Logger.Warnf("resource group %s not found, returning empty virtual networks list", resourceGroup)
			return []core.IntegrationResource{}, nil
		}
		ctx.Logger.WithError(err).Errorf("failed to list virtual networks for resourceGroup=%s", resourceGroup)
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
	ctx.Logger.Infof("listing Azure subnets for resourceGroup=%s vnet=%s", resourceGroup, vnetName)
	if resourceGroup == "" || vnetName == "" {
		return []core.IntegrationResource{}, nil
	}
	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return nil, err
	}
	return listSubnets(ctx, provider, resourceGroup, vnetName)
}

func listSubnets(ctx core.ListResourcesContext, provider *AzureProvider, resourceGroup, vnetName string) ([]core.IntegrationResource, error) {
	resourceGroup = azureResourceName(resourceGroup)
	vnetName = azureResourceName(vnetName)

	listURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets?api-version=%s",
		armBaseURL, provider.GetSubscriptionID(), resourceGroup, vnetName, armAPIVersionNetwork)

	items, err := provider.getClient().listAll(context.Background(), listURL)
	if err != nil {
		if isARMNotFound(err) {
			ctx.Logger.Warnf("resource group %s or vnet %s not found, returning empty subnets list", resourceGroup, vnetName)
			return []core.IntegrationResource{}, nil
		}
		ctx.Logger.WithError(err).Errorf("failed to list subnets for resourceGroup=%s vnet=%s", resourceGroup, vnetName)
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

// azureResourceName extracts the final resource name segment from an Azure resource ID.
// It handles plain names, full ARM IDs, and URL-encoded ARM IDs (e.g. from query parameters).
type armContainerRegistryProperties struct {
	LoginServer string `json:"loginServer"`
}

type armContainerRegistry struct {
	Name       string                          `json:"name"`
	ID         string                          `json:"id"`
	Location   string                          `json:"location"`
	Properties *armContainerRegistryProperties `json:"properties"`
}

func (a *AzureIntegration) ListContainerRegistries(ctx core.ListResourcesContext, resourceGroup string) ([]core.IntegrationResource, error) {
	ctx.Logger.Infof("listing Azure container registries for resource group: %s", resourceGroup)
	provider, err := newProvider(ctx.Integration)
	if err != nil {
		return nil, err
	}
	return listContainerRegistries(provider, resourceGroup)
}

func listContainerRegistries(provider *AzureProvider, resourceGroup string) ([]core.IntegrationResource, error) {
	var listURL string
	if resourceGroup != "" {
		listURL = fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerRegistry/registries?api-version=2023-07-01",
			armBaseURL, provider.GetSubscriptionID(), resourceGroup)
	} else {
		listURL = fmt.Sprintf("%s/subscriptions/%s/providers/Microsoft.ContainerRegistry/registries?api-version=2023-07-01",
			armBaseURL, provider.GetSubscriptionID())
	}

	items, err := provider.getClient().listAll(context.Background(), listURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list container registries: %w", err)
	}

	resources := []core.IntegrationResource{}
	for _, raw := range items {
		var reg armContainerRegistry
		if err := json.Unmarshal(raw, &reg); err != nil {
			continue
		}
		if reg.Name == "" {
			continue
		}

		id := reg.Name
		if reg.ID != "" {
			id = reg.ID
		}

		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeContainerRegistryDropdown,
			Name: reg.Name,
			ID:   id,
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	return resources, nil
}

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
