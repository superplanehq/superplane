package azure

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/sirupsen/logrus"
)

const (
	AzureFederatedTokenFileEnv = "AZURE_FEDERATED_TOKEN_FILE"

	ResourceProviderNetwork = "Microsoft.Network"
	ResourceProviderCompute = "Microsoft.Compute"

	IPAllocationMethodDynamic = "Dynamic"
	IPAllocationMethodStatic  = "Static"

	SKUStandard = "Standard"
)

type AzureProvider struct {
	credential              azcore.TokenCredential
	subscriptionID          string
	computeClient           *armcompute.VirtualMachinesClient
	networkInterfacesClient *armnetwork.InterfacesClient
	publicIPClient          *armnetwork.PublicIPAddressesClient
	resourceGroupsClient    *armresources.ResourceGroupsClient
	resourceSKUsClient      *armcompute.ResourceSKUsClient
	virtualNetworksClient   *armnetwork.VirtualNetworksClient
	subnetsClient           *armnetwork.SubnetsClient
	logger                  *logrus.Entry
}

// NewAzureProvider creates an authenticated Azure provider using federated OIDC.
func NewAzureProvider(ctx context.Context, tenantID, clientID, subscriptionID string, logger *logrus.Entry) (*AzureProvider, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required")
	}

	if clientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	tokenFilePath := os.Getenv(AzureFederatedTokenFileEnv)
	if tokenFilePath == "" {
		return nil, fmt.Errorf("environment variable %s is not set", AzureFederatedTokenFileEnv)
	}

	getAssertion := func(ctx context.Context) (string, error) {
		tokenBytes, err := os.ReadFile(tokenFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to read OIDC token from %s: %w", tokenFilePath, err)
		}

		token := string(tokenBytes)
		if token == "" {
			return "", fmt.Errorf("OIDC token file at %s is empty", tokenFilePath)
		}

		return token, nil
	}

	_, err := getAssertion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read OIDC token: %w", err)
	}

	credential, err := azidentity.NewClientAssertionCredential(tenantID, clientID, getAssertion, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create client assertion credential: %w", err)
	}

	computeClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client: %w", err)
	}

	networkInterfacesClient, err := armnetwork.NewInterfacesClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create network interfaces client: %w", err)
	}

	publicIPClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create public IP addresses client: %w", err)
	}

	resourceGroupsClient, err := armresources.NewResourceGroupsClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource groups client: %w", err)
	}

	resourceSKUsClient, err := armcompute.NewResourceSKUsClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource SKUs client: %w", err)
	}

	virtualNetworksClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual networks client: %w", err)
	}

	subnetsClient, err := armnetwork.NewSubnetsClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subnets client: %w", err)
	}

	provider := &AzureProvider{
		credential:              credential,
		subscriptionID:          subscriptionID,
		computeClient:           computeClient,
		networkInterfacesClient: networkInterfacesClient,
		publicIPClient:          publicIPClient,
		resourceGroupsClient:    resourceGroupsClient,
		resourceSKUsClient:      resourceSKUsClient,
		virtualNetworksClient:   virtualNetworksClient,
		subnetsClient:           subnetsClient,
		logger:                  logger,
	}

	logger.Infof("Azure provider initialized for subscription %s", subscriptionID)

	return provider, nil
}

// GetCredential returns the Azure token credential.
func (p *AzureProvider) GetCredential() azcore.TokenCredential {
	return p.credential
}

// GetComputeClient returns the VM client.
func (p *AzureProvider) GetComputeClient() *armcompute.VirtualMachinesClient {
	return p.computeClient
}

// GetNetworkInterfacesClient returns the NIC client.
func (p *AzureProvider) GetNetworkInterfacesClient() *armnetwork.InterfacesClient {
	return p.networkInterfacesClient
}

// GetPublicIPClient returns the Public IP client.
func (p *AzureProvider) GetPublicIPClient() *armnetwork.PublicIPAddressesClient {
	return p.publicIPClient
}

// GetResourceGroupsClient returns the Resource Groups client.
func (p *AzureProvider) GetResourceGroupsClient() *armresources.ResourceGroupsClient {
	return p.resourceGroupsClient
}

// GetResourceSKUsClient returns the Resource SKUs client.
func (p *AzureProvider) GetResourceSKUsClient() *armcompute.ResourceSKUsClient {
	return p.resourceSKUsClient
}

// GetVirtualNetworksClient returns the VNet client.
func (p *AzureProvider) GetVirtualNetworksClient() *armnetwork.VirtualNetworksClient {
	return p.virtualNetworksClient
}

// GetSubnetsClient returns the Subnets client.
func (p *AzureProvider) GetSubnetsClient() *armnetwork.SubnetsClient {
	return p.subnetsClient
}

// GetSubscriptionID returns the Azure subscription ID.
func (p *AzureProvider) GetSubscriptionID() string {
	return p.subscriptionID
}
