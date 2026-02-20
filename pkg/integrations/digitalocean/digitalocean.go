package digitalocean

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("digitalocean", &DigitalOcean{})
}

type DigitalOcean struct{}

type Configuration struct {
	APIToken string `json:"apiToken"`
}

type Metadata struct {
	AccountEmail string `json:"accountEmail"`
	AccountUUID  string `json:"accountUUID"`
}

func (d *DigitalOcean) Name() string {
	return "digitalocean"
}

func (d *DigitalOcean) Label() string {
	return "DigitalOcean"
}

func (d *DigitalOcean) Icon() string {
	return "digitalocean"
}

func (d *DigitalOcean) Description() string {
	return "Manage and monitor your DigitalOcean infrastructure"
}

func (d *DigitalOcean) Instructions() string {
	return `## Create a DigitalOcean Personal Access Token

1. Open the [DigitalOcean API Tokens page](https://cloud.digitalocean.com/account/api/tokens)
2. Click **Generate New Token**
3. Configure the token:
   - **Token name**: SuperPlane Integration
   - **Expiration**: No expiry (or choose an appropriate expiration)
   - **Scopes**: Select **Full Access** (or customize as needed)
4. Click **Generate Token**
5. Copy the token and paste it below

> **Note**: The token is only shown once. Store it securely if needed elsewhere.`
}

func (d *DigitalOcean) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "DigitalOcean Personal Access Token",
		},
	}
}

func (d *DigitalOcean) Components() []core.Component {
	return []core.Component{
		&CreateDroplet{},
	}
}

func (d *DigitalOcean) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (d *DigitalOcean) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	account, err := client.GetAccount()
	if err != nil {
		return fmt.Errorf("error fetching account: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		AccountEmail: account.Email,
		AccountUUID:  account.UUID,
	})
	ctx.Integration.Ready()
	return nil
}

func (d *DigitalOcean) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (d *DigitalOcean) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "region":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		regions, err := client.ListRegions()
		if err != nil {
			return nil, fmt.Errorf("error listing regions: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(regions))
		for _, region := range regions {
			if region.Available {
				resources = append(resources, core.IntegrationResource{
					Type: resourceType,
					Name: region.Name,
					ID:   region.Slug,
				})
			}
		}
		return resources, nil

	case "size":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		sizes, err := client.ListSizes()
		if err != nil {
			return nil, fmt.Errorf("error listing sizes: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(sizes))
		for _, size := range sizes {
			if size.Available {
				resources = append(resources, core.IntegrationResource{
					Type: resourceType,
					Name: size.Slug,
					ID:   size.Slug,
				})
			}
		}
		return resources, nil

	case "image":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		images, err := client.ListImages("distribution")
		if err != nil {
			return nil, fmt.Errorf("error listing images: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(images))
		for _, image := range images {
			name := image.Name
			if image.Distribution != "" {
				name = fmt.Sprintf("%s %s", image.Distribution, image.Name)
			}
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: name,
				ID:   image.Slug,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}

func (d *DigitalOcean) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (d *DigitalOcean) Actions() []core.Action {
	return []core.Action{}
}

func (d *DigitalOcean) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
