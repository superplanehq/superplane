package cloudsmith

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("cloudsmith", &Cloudsmith{})
}

type Cloudsmith struct{}

type Configuration struct {
	APIKey    string `json:"apiKey"`
	Workspace string `json:"workspace"`
}

func (c *Cloudsmith) Name() string {
	return "cloudsmith"
}

func (c *Cloudsmith) Label() string {
	return "Cloudsmith"
}

func (c *Cloudsmith) Icon() string {
	return "cloudsmith"
}

func (c *Cloudsmith) Description() string {
	return "Manage and react to Cloudsmith package repositories"
}

func (c *Cloudsmith) Instructions() string {
	return `
To generate a Cloudsmith API token:
- Go to **Cloudsmith** → **Account Settings** → **API Key**
- Copy the API key and enter it below, along with your organization or user workspace (slug)
`
}

func (c *Cloudsmith) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Cloudsmith API token",
		},
		{
			Name:        "workspace",
			Label:       "Workspace",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Cloudsmith organization or user workspace (slug)",
		},
	}
}

func (c *Cloudsmith) Components() []core.Component {
	return []core.Component{&GetPackage{}}
}

func (c *Cloudsmith) Triggers() []core.Trigger {
	return []core.Trigger{&OnPackageEvent{}}
}

func (c *Cloudsmith) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *Cloudsmith) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if err := client.ValidateCredentials(); err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (c *Cloudsmith) HandleRequest(ctx core.HTTPRequestContext) {}

func (c *Cloudsmith) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return listCloudsmithResources(resourceType, ctx)
}

func (c *Cloudsmith) Actions() []core.Action {
	return []core.Action{}
}

func (c *Cloudsmith) HandleAction(ctx core.IntegrationActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}
