package azure

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	integration := &AzureIntegration{}
	registry.RegisterIntegrationWithWebhookHandler("azure", integration, &AzureWebhookHandler{integration: integration})
}

type AzureIntegration struct {
	provider *AzureProvider
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
	}
}

func (a *AzureIntegration) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnVMCreatedTrigger{},
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
	oidcProvider := ctx.OIDC

	getAssertion := func(_ context.Context) (string, error) {
		subject := fmt.Sprintf("app-installation:%s", integrationID)
		return oidcProvider.Sign(subject, 5*time.Minute, integrationID, nil)
	}

	provider, err := NewAzureProvider(
		context.Background(),
		config.TenantID,
		config.ClientID,
		config.SubscriptionID,
		getAssertion,
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize Azure provider: %w", err)
	}

	a.provider = provider

	// Verify credentials by listing resource groups.
	// This proves that the federated identity credential is configured correctly.
	verifyURL := fmt.Sprintf("%s/subscriptions/%s/resourcegroups?api-version=%s",
		armBaseURL, config.SubscriptionID, armAPIVersionResources)

	_, err = provider.GetClient().listAll(context.Background(), verifyURL)
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

	case ResourceTypeVMSizeDropdown:
		return a.ListVMSizes(ctx, firstNonEmptyParameter(ctx.Parameters, "location"))

	case ResourceTypeVirtualNetworkDropdown:
		return a.ListVirtualNetworks(ctx, firstNonEmptyParameter(ctx.Parameters, "resourceGroup"))

	case ResourceTypeSubnetDropdown:
		return a.ListSubnets(
			ctx,
			firstNonEmptyParameter(ctx.Parameters, "resourceGroup"),
			firstNonEmptyParameter(ctx.Parameters, "virtualNetworkName", "virtualNetwork", "vnetName"),
		)

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

// GetProvider returns the initialized provider.
func (a *AzureIntegration) GetProvider() *AzureProvider {
	return a.provider
}
