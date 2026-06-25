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
	APIKey string `json:"apiKey"`
}

type Metadata struct {
	UserSlug  string `json:"userSlug"`
	UserName  string `json:"userName"`
	UserEmail string `json:"userEmail"`
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
	return "Automate repository, package, and artifact management on Cloudsmith"
}

func (c *Cloudsmith) Instructions() string {
	return `## Cloudsmith Service Account API Key

SuperPlane authenticates to Cloudsmith using a service account API key, which is not tied to an individual user.

1. In the Cloudsmith web dashboard, go to the **Accounts** tab and click on **Services**
2. Click on **New Service**. Give the service a name like **Superplane** and optional description. Assign the **Manager** role to the service.
3. Click on **Create Service** and copy the generated API key.
4. Paste the API key below.
5. To give the service access to any repository, click on your Repository and then **Settings** → **Access control → Privileges for specific services**, and add the service with the **Admin** privilege.

> **Note:** The **Promote Package** action (copy or move a package between repositories) requires **Admin** privilege on **both** the source and destination repositories. Make sure the service account has been granted Admin access to every repository it needs to promote packages to or from.
`
}

func (c *Cloudsmith) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Cloudsmith service account API key",
		},
	}
}

func (c *Cloudsmith) Actions() []core.Action {
	return []core.Action{
		&GetRepository{},
		&GetPackage{},
		&ResyncPackage{},
		&TagPackage{},
		&DeletePackage{},
		&ListPackages{},
		&PromotePackage{},
	}
}

func (c *Cloudsmith) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnSecurityScanCompleted{},
		&OnPackageCreated{},
	}
}

func (c *Cloudsmith) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	user, err := client.GetSelf()
	if err != nil {
		return fmt.Errorf("error validating API key: %v", err)
	}

	if !user.Authenticated {
		return fmt.Errorf("API key is not valid")
	}

	ctx.Integration.SetMetadata(Metadata{
		UserSlug:  user.Slug,
		UserName:  user.Name,
		UserEmail: user.Email,
	})
	ctx.Integration.Ready()
	return nil
}

func (c *Cloudsmith) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *Cloudsmith) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (c *Cloudsmith) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *Cloudsmith) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
