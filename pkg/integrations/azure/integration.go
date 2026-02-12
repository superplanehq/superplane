package azure

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("azure", &AzureIntegration{}, &AzureWebhookHandler{})
}

type AzureIntegration struct {
	provider *AzureProvider
}

type Configuration struct {
	TenantID       string `json:"tenantId" mapstructure:"tenantId"`
	ClientID       string `json:"clientId" mapstructure:"clientId"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

type Metadata struct {
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

### 2. Configure Federated Identity Credential

1. In your app registration, go to **Certificates & secrets** → **Federated credentials**
2. Click **Add credential**
3. Select **Other issuer**
4. Configure the credential:
   - **Issuer**: The SuperPlane OIDC issuer URL (provided after creation)
   - **Subject identifier**: ` + "`app-installation:<integration-id>`" + ` (provided after creation)
   - **Audience**: The integration ID (provided after creation)
   - **Name**: ` + "`superplane-integration`" + ` (or any descriptive name)

### 3. Grant Required Permissions

Assign appropriate Azure RBAC roles to your app registration:

- **Virtual Machine Contributor** - For VM management
- **Network Contributor** - For network resource management
- **Storage Account Contributor** - For storage operations (if needed)
- **EventGrid Contributor** - For Event Grid subscriptions

You can assign these roles at the subscription or resource group level.

### 4. Complete the Connection

Enter the following information below:
- **Tenant ID**: Your Azure AD tenant ID
- **Client ID**: Your app registration's client ID
- **Subscription ID**: Your Azure subscription ID

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
		&CreateVMComponent{},
	}
}

func (a *AzureIntegration) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnVMCreatedTrigger{},
	}
}

// Sync validates configuration and initializes Azure clients.
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

	provider, err := NewAzureProvider(
		context.Background(),
		config.TenantID,
		config.ClientID,
		config.SubscriptionID,
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize Azure provider: %w", err)
	}

	a.provider = provider

	ctx.Logger.Info("Azure integration synchronized successfully")

	ctx.Integration.Ready()

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

// HandleRequest routes incoming webhook requests.
func (a *AzureIntegration) HandleRequest(ctx core.HTTPRequestContext) {
	if ctx.Request.Method == http.MethodPost {
		if strings.HasSuffix(ctx.Request.URL.Path, "/webhook") ||
			strings.HasSuffix(ctx.Request.URL.Path, "/events") {
			a.handleWebhook(ctx)
			return
		}
	}

	ctx.Logger.Warnf("Unknown request path: %s %s", ctx.Request.Method, ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)
	ctx.Response.Write([]byte("not found"))
}

// handleWebhook processes Azure Event Grid webhooks.
func (a *AzureIntegration) handleWebhook(ctx core.HTTPRequestContext) {
	ctx.Logger.Infof("Handling Azure Event Grid webhook: %s", ctx.Request.URL.Path)

	if err := HandleWebhook(ctx.Response, ctx.Request, ctx.Logger); err != nil {
		ctx.Logger.Errorf("Failed to handle webhook: %v", err)
		return
	}

	ctx.Logger.Info("Webhook processed successfully")
}

// GetProvider returns the initialized provider.
func (a *AzureIntegration) GetProvider() *AzureProvider {
	return a.provider
}
