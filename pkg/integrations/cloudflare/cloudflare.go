package cloudflare

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("cloudflare", &Cloudflare{})
}

type Cloudflare struct{}

type Configuration struct {
	APIToken string `json:"apiToken"`
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
     - Zone / Dynamic Redirect / Edit
   - **Zone Resources**: Include / All zones _(or select specific zones)_
5. Click **Continue to summary**, then **Create Token**
6. Copy the token and paste it below

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
	}
}

func (c *Cloudflare) Components() []core.Component {
	return []core.Component{
		&UpdateRedirectRule{},
	}
}

func (c *Cloudflare) Triggers() []core.Trigger {
	return []core.Trigger{}
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

	default:
		return []core.IntegrationResource{}, nil
	}
}

func (c *Cloudflare) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (c *Cloudflare) CompareWebhookConfig(a, b any) (bool, error) {
	return false, nil
}

func (c *Cloudflare) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (c *Cloudflare) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
