package notion

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To configure Notion to work with SuperPlane:

1. **Create an internal integration**: In Notion, open [My integrations](https://www.notion.so/my-integrations) and create a new internal integration for your workspace.
2. **Copy the token**: Copy the integration's internal integration secret.
3. **Share a database**: Open the database SuperPlane should watch, choose **Connections** from the page menu, and add your integration.
4. **Enter the token**: Provide the token in the integration configuration.
`

func init() {
	// No webhook handler: Notion has no webhook API for internal
	// integrations, so the onPageAdded trigger polls instead.
	registry.RegisterIntegration("notion", &Notion{})
}

type Notion struct{}

func (n *Notion) Name() string {
	return "notion"
}

func (n *Notion) Label() string {
	return "Notion"
}

func (n *Notion) Icon() string {
	return "notion"
}

func (n *Notion) Description() string {
	return "Create Backlog work orders from new Notion pages"
}

func (n *Notion) Instructions() string {
	return installationInstructions
}

func (n *Notion) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "Internal Integration Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Internal integration token from Notion > My integrations",
		},
	}
}

func (n *Notion) Actions() []core.Action {
	return []core.Action{}
}

func (n *Notion) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPageAdded{},
	}
}

func (n *Notion) Sync(ctx core.SyncContext) error {
	config := struct {
		APIToken string `mapstructure:"apiToken"`
	}{}

	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if strings.TrimSpace(config.APIToken) == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.ValidateCredentials(); err != nil {
		return fmt.Errorf("invalid credentials: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (n *Notion) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (n *Notion) Hooks() []core.Hook {
	return []core.Hook{}
}

func (n *Notion) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}

func (n *Notion) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op - Notion does not call SuperPlane; the onPageAdded trigger polls.
}

func (n *Notion) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeDatabase:
		return listDatabaseResources(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listDatabaseResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	databases, err := client.ListDatabases()
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(databases))
	for _, database := range databases {
		name := database.Name
		if name == "" {
			name = "Untitled database"
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeDatabase,
			Name: name,
			ID:   database.ID,
		})
	}

	return resources, nil
}
