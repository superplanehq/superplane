package flyio

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("flyio", &FlyIO{})
}

type FlyIO struct{}

type Configuration struct {
	OrgSlug string `json:"orgSlug" mapstructure:"orgSlug"`
}

type Metadata struct {
	Apps []App `json:"apps"`
}

func (f *FlyIO) Name() string {
	return "flyio"
}

func (f *FlyIO) Label() string {
	return "Fly.io"
}

func (f *FlyIO) Icon() string {
	return "server"
}

func (f *FlyIO) Description() string {
	return "Deploy and manage applications, Machines, and Volumes on Fly.io"
}

func (f *FlyIO) Instructions() string {
	return `## Create a Fly.io API Token

1. Go to your [Fly.io Dashboard](https://fly.io/dashboard) and click on your organization
2. Navigate to **Access Tokens** in the left sidebar
3. Click **Create Token** and give it a descriptive name like "SuperPlane Integration"
4. Copy the token and paste it below

Alternatively, you can create a token using the Fly CLI:

` + "```bash" + `
# For app-scoped deployments
fly tokens deploy

# For organization-wide access
fly tokens create
` + "```" + `

> **Important**: Store the token securely. It will only be shown once.

## Organization Slug

Enter your organization slug (e.g., "personal" for personal accounts). This will be used as the default organization for listing apps and creating new ones.`
}

func (f *FlyIO) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Fly.io API token with appropriate permissions",
		},
		{
			Name:        "orgSlug",
			Label:       "Organization Slug",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Default organization slug (e.g., 'personal'). Used for listing apps and creating new ones.",
		},
	}
}

func (f *FlyIO) Components() []core.Component {
	return []core.Component{
		&ListApps{},
	}
}

func (f *FlyIO) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAppStateChange{},
	}
}

func (f *FlyIO) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// List apps to validate the token and populate metadata
	orgSlug := config.OrgSlug
	if orgSlug == "" {
		orgSlug = "personal"
	}

	apps, err := client.ListApps(orgSlug)
	if err != nil {
		return fmt.Errorf("error listing apps: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{Apps: apps})
	ctx.Integration.Ready()
	return nil
}

func (f *FlyIO) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (f *FlyIO) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "app":
		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode application metadata: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(metadata.Apps))
		for _, app := range metadata.Apps {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: app.Name,
				ID:   app.Name, // Fly.io uses app name as the primary identifier
			})
		}
		return resources, nil

	case "machine":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode application metadata: %w", err)
		}

		var resources []core.IntegrationResource
		for _, app := range metadata.Apps {
			machines, err := client.ListMachines(app.Name)
			if err != nil {
				continue
			}

			for _, machine := range machines {
				resources = append(resources, core.IntegrationResource{
					Type: resourceType,
					Name: fmt.Sprintf("%s - %s (%s)", app.Name, machine.Name, machine.ID),
					ID:   fmt.Sprintf("%s/%s", app.Name, machine.ID),
				})
			}
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}

func (f *FlyIO) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (f *FlyIO) CompareWebhookConfig(a, b any) (bool, error) {
	return false, nil
}

func (f *FlyIO) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (f *FlyIO) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}

func (f *FlyIO) Actions() []core.Action {
	return []core.Action{}
}

func (f *FlyIO) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
