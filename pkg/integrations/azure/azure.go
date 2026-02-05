package azure

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/azure/common"
	"github.com/superplanehq/superplane/pkg/integrations/azure/compute"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	defaultTokenValidityMinutes = 60
	minTokenValidityMinutes     = 15
	maxTokenValidityMinutes     = 120
	oidcRequestLifetime         = 5 * time.Minute
)

func init() {
	registry.RegisterIntegration("azure", &Azure{})
}

type Azure struct{}

type Configuration struct {
	TenantID               string       `json:"tenantId" mapstructure:"tenantId"`
	SubscriptionID         string       `json:"subscriptionId" mapstructure:"subscriptionId"`
	ClientID               string       `json:"clientId" mapstructure:"clientId"`
	Location               string       `json:"location" mapstructure:"location"`
	TokenValidityMinutes   int          `json:"tokenValidityMinutes" mapstructure:"tokenValidityMinutes"`
	Tags                   []common.Tag `json:"tags" mapstructure:"tags"`
}

func (a *Azure) Name() string {
	return "azure"
}

func (a *Azure) Label() string {
	return "Azure"
}

func (a *Azure) Icon() string {
	return "azure"
}

func (a *Azure) Description() string {
	return "Manage Azure resources and run workflows against your subscription"
}

func (a *Azure) Instructions() string {
	return "Create an App Registration in Azure AD with a federated credential (OIDC) pointing at SuperPlane. Leave the fields empty to see step-by-step instructions."
}

func (a *Azure) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "tenantId",
			Label:       "Tenant ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Azure AD tenant (directory) ID",
		},
		{
			Name:        "subscriptionId",
			Label:       "Subscription ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Azure subscription ID for resource management",
		},
		{
			Name:        "clientId",
			Label:       "Application (client) ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Application ID of the Azure AD App Registration",
		},
		{
			Name:        "location",
			Label:       "Default location",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "eastus",
			Description: "Default Azure region for resources (e.g. eastus, westus2)",
		},
		{
			Name:        "tokenValidityMinutes",
			Label:       "Token validity (minutes)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     fmt.Sprintf("%d", defaultTokenValidityMinutes),
			Description: "Requested access token lifetime (capped by Azure AD policy)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := minTokenValidityMinutes; return &min }(),
					Max: func() *int { max := maxTokenValidityMinutes; return &max }(),
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Tags to apply to Azure resources created by this integration",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:               "key",
								Label:              "Key",
								Type:               configuration.FieldTypeString,
								Required:           true,
								DisallowExpression: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func (a *Azure) Components() []core.Component {
	return []core.Component{
		&compute.CreateVirtualMachine{},
	}
}

func (a *Azure) Triggers() []core.Trigger {
	return nil
}

func (a *Azure) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	metadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	tenantID := strings.TrimSpace(config.TenantID)
	subscriptionID := strings.TrimSpace(config.SubscriptionID)
	clientID := strings.TrimSpace(config.ClientID)
	if tenantID == "" || subscriptionID == "" || clientID == "" {
		return a.showBrowserAction(ctx)
	}

	metadata.Tags = common.NormalizeTags(config.Tags)

	tokenResult, err := ObtainToken(
		ctx.HTTP,
		ctx.OIDC,
		tenantID,
		clientID,
		ctx.Integration.ID().String(),
		oidcRequestLifetime,
	)
	if err != nil {
		return fmt.Errorf("failed to obtain Azure AD token: %w", err)
	}

	if err := ctx.Integration.SetSecret("accessToken", []byte(tokenResult.AccessToken)); err != nil {
		return fmt.Errorf("failed to store access token: %w", err)
	}

	location := strings.TrimSpace(config.Location)
	if location == "" {
		location = "eastus"
	}

	metadata.Session = &common.SessionMetadata{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		ClientID:       clientID,
		ExpiresAt:      tokenResult.ExpiresAt.Format(time.RFC3339),
		Location:       location,
	}

	refreshAfter := time.Until(tokenResult.ExpiresAt) / 2
	if refreshAfter < time.Minute {
		refreshAfter = time.Minute
	}
	if err := ctx.Integration.ScheduleResync(refreshAfter); err != nil {
		ctx.Logger.Warnf("failed to schedule Azure token resync: %v", err)
	}

	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.Ready()
	ctx.Integration.RemoveBrowserAction()

	return nil
}

func (a *Azure) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (a *Azure) showBrowserAction(ctx core.SyncContext) error {
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: fmt.Sprintf(`
**1. Create an App Registration**

- Go to Azure Portal → Microsoft Entra ID (Azure AD) → App registrations → New registration
- Name it (e.g. "SuperPlane") and select the appropriate account type
- Do **not** create a client secret

**2. Add a Federated credential (OIDC)**

- Open the app → Certificates & secrets → Federated credentials → Add credential
- Federated credential scenario: **Other**
- Issuer: **%s**
- Subject identifier: **app-installation:%s**
- Audience: **%s**
- Name the credential (e.g. "superplane") and save

**3. Grant API permissions**

- Go to API permissions → Add permission → Microsoft Graph (if needed) and **Microsoft Azure Management**
- Under Azure Management, add **Application permission**: e.g. appropriate scope for the resources you will manage (e.g. Virtual Machines)
- Grant admin consent if required

**4. Complete the installation**

- Copy **Tenant ID** (Overview), **Application (client) ID** (Overview), and **Subscription ID** (Subscriptions in Azure Portal)
- Paste them into the integration configuration
`, ctx.BaseURL, ctx.Integration.ID().String(), ctx.Integration.ID().String()),
	})
	return nil
}

func (a *Azure) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.Response.WriteHeader(404)
}

func (a *Azure) CompareWebhookConfig(_, _ any) (bool, error) {
	return false, nil
}

func (a *Azure) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (a *Azure) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}

func (a *Azure) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "compute.virtualMachine":
		return compute.ListVirtualMachines(ctx, resourceType)
	default:
		return nil, nil
	}
}

func (a *Azure) Actions() []core.Action {
	return nil
}

func (a *Azure) HandleAction(ctx core.IntegrationActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}
