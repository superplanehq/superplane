package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/sirupsen/logrus"
)

type AzureProvider struct {
	credential     azcore.TokenCredential
	subscriptionID string
	client         *armClient
	logger         *logrus.Entry
}

// NewAzureProvider creates an authenticated Azure provider using a caller-supplied
// OIDC assertion callback. The getAssertion function is called by the Azure SDK
// each time a new access token is needed.
func NewAzureProvider(ctx context.Context, tenantID, clientID, subscriptionID string, getAssertion func(context.Context) (string, error), logger *logrus.Entry) (*AzureProvider, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required")
	}
	if clientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	// Verify the assertion callback works at startup
	if _, err := getAssertion(ctx); err != nil {
		return nil, fmt.Errorf("failed to obtain OIDC assertion: %w", err)
	}

	credential, err := azidentity.NewClientAssertionCredential(tenantID, clientID, getAssertion, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create client assertion credential: %w", err)
	}

	provider := &AzureProvider{
		credential:     credential,
		subscriptionID: subscriptionID,
		client:         newARMClient(credential, subscriptionID, logger),
		logger:         logger,
	}

	logger.Infof("Azure provider initialized for subscription %s", subscriptionID)
	return provider, nil
}

// GetCredential returns the Azure token credential.
func (p *AzureProvider) GetCredential() azcore.TokenCredential {
	return p.credential
}

// getClient returns the ARM REST client.
func (p *AzureProvider) getClient() *armClient {
	return p.client
}

// GetSubscriptionID returns the Azure subscription ID.
func (p *AzureProvider) GetSubscriptionID() string {
	return p.subscriptionID
}
