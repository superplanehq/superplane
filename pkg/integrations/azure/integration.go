package azure

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/oidc"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	integration := &AzureIntegration{}
	registry.RegisterIntegrationWithWebhookHandler("azure", integration, &AzureWebhookHandler{integration: integration})
}

type AzureIntegration struct {
	mu       sync.Mutex
	provider *AzureProvider

	// oidcProvider is populated during Sync from SyncContext.OIDC and
	// reused by ensureProvider to lazily create Azure clients.
	oidcProvider oidc.Provider

	// Cached from the last ensureProvider / Sync call so the provider
	// can be reused across requests without re-reading the DB.
	integrationID string
	config        *Configuration
}

type Configuration struct {
	TenantID       string `json:"tenantId" mapstructure:"tenantId"`
	ClientID       string `json:"clientId" mapstructure:"clientId"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

func (a *AzureIntegration) Name() string {
	return "azure"
}

func (a *AzureIntegration) Label() string {
	return "Microsoft Azure"
}

func (a *AzureIntegration) Icon() string {
	return "azure"
}

func (a *AzureIntegration) Description() string {
	return "Manage and automate Microsoft Azure resources and services"
}

func (a *AzureIntegration) Instructions() string {
	return `## Azure Workload Identity Federation Setup

To connect SuperPlane to Microsoft Azure using Workload Identity Federation:

### 1. Create or Select an App Registration

1. Go to **Azure Portal** → **Azure Active Directory** → **App registrations**
2. Create a new registration or select an existing app
3. Note the **Application (client) ID** and **Directory (tenant) ID**

### 2. Complete the Connection

Enter the following information below and create the integration:
- **Tenant ID**: Your Azure AD tenant ID
- **Client ID**: Your app registration's client ID
- **Subscription ID**: Your Azure subscription ID

After creation, you will be guided through configuring the Federated Identity Credential and granting the required permissions.

SuperPlane will use Workload Identity Federation to authenticate without storing any credentials.`
}

func (a *AzureIntegration) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "tenantId",
			Label:       "Tenant ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Azure Active Directory tenant ID (Directory ID)",
			Placeholder: "00000000-0000-0000-0000-000000000000",
		},
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Application (client) ID from your Azure app registration",
			Placeholder: "00000000-0000-0000-0000-000000000000",
		},
		{
			Name:        "subscriptionId",
			Label:       "Subscription ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Azure subscription ID where resources will be managed",
			Placeholder: "00000000-0000-0000-0000-000000000000",
		},
	}
}

func (a *AzureIntegration) Components() []core.Component {
	return []core.Component{
		&CreateVMComponent{integration: a},
		&DeleteVMComponent{integration: a},
	}
}

func (a *AzureIntegration) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnVMDeleted{integration: a},
		&OnImagePushed{integration: a},
		&OnImageDeleted{integration: a},
	}
}

// Sync validates configuration, initializes Azure clients, and verifies credentials.
// On the first sync the federated identity credential is typically not yet configured
// in Azure AD, so the verification call will fail and a BrowserAction with setup
// instructions is shown. Once the user configures the credential and re-syncs,
// verification succeeds and the integration transitions to Ready.
func (a *AzureIntegration) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.TenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}

	if config.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}

	if config.SubscriptionID == "" {
		return fmt.Errorf("subscription ID is required")
	}

	ctx.Logger.Infof("Initializing Azure provider: tenant=%s, subscription=%s",
		config.TenantID, config.SubscriptionID)

	integrationID := ctx.Integration.ID().String()

	a.mu.Lock()
	a.oidcProvider = ctx.OIDC
	a.integrationID = integrationID
	a.config = &config

	provider, err := a.initProviderLocked()
	if err != nil {
		a.mu.Unlock()
		return fmt.Errorf("failed to initialize Azure provider: %w", err)
	}
	a.provider = provider
	a.mu.Unlock()

	// Verify credentials by listing resource groups.
	// This proves that the federated identity credential is configured correctly.
	verifyURL := fmt.Sprintf("%s/subscriptions/%s/resourcegroups?api-version=%s",
		armBaseURL, config.SubscriptionID, armAPIVersionResources)

	_, err = provider.getClient().listAll(context.Background(), verifyURL)
	if err != nil {
		ctx.Logger.Infof("Credential verification failed: %v", err)
		ctx.Logger.Info("Showing setup instructions for federated identity credential")
		return a.showBrowserAction(ctx)
	}

	ctx.Integration.Ready()
	ctx.Integration.RemoveBrowserAction()
	ctx.Logger.Info("Azure integration synchronized successfully")

	return nil
}

func (a *AzureIntegration) showBrowserAction(ctx core.SyncContext) error {
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: fmt.Sprintf(`
**1. Configure Federated Identity Credential**

- In your app registration, go to **Certificates & secrets** → **Federated credentials**
- Click **Add credential** → Select **Other issuer**
- Issuer: **%s**
- Subject identifier: **app-installation:%s**
- Audience: **%s**
- Name: **superplane-integration** (or any descriptive name)

**2. Grant Required Permissions**

Assign Azure RBAC roles to your app registration at the subscription or resource group level:

- **Virtual Machine Contributor** – For VM management
- **Network Contributor** – For network resource management
- **EventGrid Contributor** – For Event Grid subscriptions

**3. Complete Setup**

After configuring the credential and permissions above, click **Save** to re-sync the integration. SuperPlane will verify the connection automatically.
`, ctx.WebhooksBaseURL, ctx.Integration.ID().String(), ctx.Integration.ID().String()),
	})

	return nil
}

// Cleanup handles integration teardown.
func (a *AzureIntegration) Cleanup(ctx core.IntegrationCleanupContext) error {
	ctx.Logger.Info("Cleaning up Azure integration")
	return nil
}

// Actions returns integration-level actions.
func (a *AzureIntegration) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction executes an integration-level action.
func (a *AzureIntegration) HandleAction(ctx core.IntegrationActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

// ListResources lists Azure resources by resource type.
func (a *AzureIntegration) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeResourceGroupDropdown:
		return a.ListResourceGroups(ctx)

	case ResourceTypeResourceGroupLocation:
		return a.ListResourceGroupLocations(ctx, firstNonEmptyParameter(ctx.Parameters, "resourceGroup"))

	case ResourceTypeVMSizeDropdown:
		return a.ListVMSizes(ctx, firstNonEmptyParameter(ctx.Parameters, "resourceGroup"))

	case ResourceTypeVirtualNetworkDropdown:
		return a.ListVirtualNetworks(ctx, firstNonEmptyParameter(ctx.Parameters, "resourceGroup"))

	case ResourceTypeSubnetDropdown:
		return a.ListSubnets(
			ctx,
			firstNonEmptyParameter(ctx.Parameters, "resourceGroup"),
			firstNonEmptyParameter(ctx.Parameters, "virtualNetworkName", "virtualNetwork", "vnetName"),
		)

	case ResourceTypeContainerRegistryDropdown:
		return a.ListContainerRegistries(ctx, firstNonEmptyParameter(ctx.Parameters, "resourceGroup"))

	case "resourceGroup", "virtualNetwork", "subnet":
		return []core.IntegrationResource{}, nil

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func firstNonEmptyParameter(parameters map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := parameters[key]; ok && value != "" {
			return value
		}
	}
	return ""
}

// HandleRequest routes incoming HTTP requests.
func (a *AzureIntegration) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.Logger.Warnf("Unknown request path: %s %s", ctx.Request.Method, ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)
	ctx.Response.Write([]byte("not found"))
}

// ensureProvider returns the cached Azure provider, or lazily creates one
// by reading the integration configuration from the database and using the
// OIDC provider stored during Sync. This is the single entry point for all
// code paths that need an authenticated Azure client (ListResources,
// Execute, WebhookHandler).
func (a *AzureIntegration) ensureProvider(integrationCtx core.IntegrationContext) (*AzureProvider, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	integrationID := integrationCtx.ID().String()

	// Return the cached provider when it matches the current integration.
	if a.provider != nil && a.integrationID == integrationID {
		return a.provider, nil
	}

	if a.oidcProvider == nil {
		return nil, fmt.Errorf("Azure OIDC provider not available; server may not have finished starting")
	}

	config, err := loadConfig(integrationCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to load Azure configuration: %w", err)
	}

	a.integrationID = integrationID
	a.config = config

	provider, err := a.initProviderLocked()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Azure provider: %w", err)
	}

	a.provider = provider
	return a.provider, nil
}

// loadConfig reads the Azure integration configuration from the database
// through the IntegrationContext.
func loadConfig(ctx core.IntegrationContext) (*Configuration, error) {
	tenantID, err := ctx.GetConfig("tenantId")
	if err != nil {
		return nil, fmt.Errorf("tenantId: %w", err)
	}

	clientID, err := ctx.GetConfig("clientId")
	if err != nil {
		return nil, fmt.Errorf("clientId: %w", err)
	}

	subscriptionID, err := ctx.GetConfig("subscriptionId")
	if err != nil {
		return nil, fmt.Errorf("subscriptionId: %w", err)
	}

	return &Configuration{
		TenantID:       string(tenantID),
		ClientID:       string(clientID),
		SubscriptionID: string(subscriptionID),
	}, nil
}

// initProviderLocked creates a new AzureProvider using the stored OIDC credentials.
// Caller must hold a.mu.
func (a *AzureIntegration) initProviderLocked() (*AzureProvider, error) {
	if a.oidcProvider == nil || a.config == nil {
		return nil, fmt.Errorf("OIDC provider or config not available")
	}

	oidcProv := a.oidcProvider
	integrationID := a.integrationID

	getAssertion := func(_ context.Context) (string, error) {
		subject := fmt.Sprintf("app-installation:%s", integrationID)
		return oidcProv.Sign(subject, 5*time.Minute, integrationID, nil)
	}

	return NewAzureProvider(
		context.Background(),
		a.config.TenantID,
		a.config.ClientID,
		a.config.SubscriptionID,
		getAssertion,
		logrus.NewEntry(logrus.StandardLogger()),
	)
}
