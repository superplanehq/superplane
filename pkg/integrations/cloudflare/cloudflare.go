package cloudflare

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("cloudflare", &Cloudflare{}, &CloudflareWebhookHandler{})
}

type Cloudflare struct{}

type Configuration struct {
	APIToken  string `json:"apiToken"`
	AccountID string `json:"accountId"`
}

type Metadata struct {
	Zones []Zone `json:"zones"`
}

func (c *Cloudflare) Name() string {
	return "cloudflare"
}

func (c *Cloudflare) Label() string {
	return "Cloudflare"
}

func (c *Cloudflare) Icon() string {
	return "cloud"
}

func (c *Cloudflare) Description() string {
	return "Manage Cloudflare zones, rules, and DNS"
}

func (c *Cloudflare) Instructions() string {
	return `## Create a Cloudflare API Token

1. Open the [Cloudflare API Tokens page](https://dash.cloudflare.com/profile/api-tokens)
2. Click **Create Token**
3. Click **Get started** next to "Create Custom Token"
4. Configure the token:
   - **Token name**: SuperPlane Integration
   - **Permissions** (click "+ Add more" to add each):
     - Zone / Zone / Read
     - Zone / DNS / Edit
     - Zone / Dynamic Redirect / Edit
     - Account / Load Balancing: Monitors and Pools / Edit
     - Account / Notifications / Edit
     - Account / Account Settings / Edit
   - **Zone Resources**: Include / All zones _(or select specific zones)_
   - **Account Resources**: Include the account containing your load balancers
5. Click **Continue to summary**, then **Create Token**
6. Copy the token and paste it below

## Find your Cloudflare Account ID

The **Account ID** is required for load balancing monitors and health alert webhooks.

1. Open the [Cloudflare dashboard](https://dash.cloudflare.com/)
2. Select the account that contains your load balancers
3. In the account home page, copy the **Account ID** from the right sidebar
4. Paste it into the **Account ID** field below

Make sure this is the same account selected in **Account Resources** when creating the API token.

> **Note**: The token is only shown once. Store it securely if needed elsewhere.`
}

func (c *Cloudflare) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Cloudflare API Token with appropriate permissions",
		},
		{
			Name:        "accountId",
			Label:       "Account ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Cloudflare account ID from the account home page right sidebar. Required for load balancing monitors and alerting webhooks.",
		},
	}
}

func (c *Cloudflare) Actions() []core.Action {
	return []core.Action{
		&CreateDNSRecord{},
		&UpdateRedirectRule{},
		&UpdateDNSRecord{},
		&DeleteDNSRecord{},
		&CreateMonitor{},
		&DeleteMonitor{},
	}
}

func (c *Cloudflare) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnLoadBalancingHealthAlert{},
	}
}

func (c *Cloudflare) Sync(ctx core.SyncContext) error {
	configuration := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &configuration)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if configuration.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	if strings.TrimSpace(configuration.AccountID) == "" {
		return fmt.Errorf("accountId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	zones, err := client.ListZones()
	if err != nil {
		return fmt.Errorf("error listing zones: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{Zones: zones})
	ctx.Integration.Ready()
	return nil
}

func (c *Cloudflare) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *Cloudflare) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "zone":
		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode application metadata: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(metadata.Zones))
		for _, zone := range metadata.Zones {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: zone.Name,
				ID:   zone.ID,
			})
		}
		return resources, nil

	case "redirect_rule":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode application metadata: %w", err)
		}

		var resources []core.IntegrationResource
		for _, zone := range metadata.Zones {
			rules, err := client.ListRedirectRules(zone.ID)
			if err != nil {
				continue
			}

			for _, rule := range rules {
				resources = append(resources, core.IntegrationResource{
					Type: resourceType,
					Name: fmt.Sprintf("%s - %s", zone.Name, rule.Description),
					ID:   fmt.Sprintf("%s/%s", zone.ID, rule.ID),
				})
			}
		}
		return resources, nil

	case "dns_record":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode application metadata: %w", err)
		}

		var resources []core.IntegrationResource
		for _, zone := range metadata.Zones {
			records, err := client.ListDNSRecords(zone.ID)
			if err != nil {
				continue
			}

			for _, record := range records {
				resources = append(resources, core.IntegrationResource{
					Type: resourceType,
					Name: fmt.Sprintf("%s (%s)", record.Name, record.Type),
					ID:   fmt.Sprintf("%s/%s", zone.ID, record.ID),
				})
			}
		}
		return resources, nil

	case "monitor":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		accountID, err := accountIDForIntegration(ctx.Integration)
		if err != nil {
			return nil, err
		}

		monitors, err := client.ListMonitors(accountID)
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(monitors))
		for _, monitor := range monitors {
			name := monitor.Description
			if name == "" {
				name = monitor.ID
			}
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: name,
				ID:   monitor.ID,
			})
		}
		return resources, nil

	case "pool":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		accountID, err := accountIDForIntegration(ctx.Integration)
		if err != nil {
			return nil, err
		}

		pools, err := client.ListPools(accountID)
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(pools))
		for _, pool := range pools {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: pool.Name,
				ID:   pool.ID,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}

func (c *Cloudflare) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (c *Cloudflare) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *Cloudflare) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
