package notion

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("notion", &Notion{})
}

type Notion struct{}

func (n *Notion) Name() string        { return "notion" }
func (n *Notion) Label() string       { return "Notion" }
func (n *Notion) Icon() string        { return "notion" }
func (n *Notion) Description() string { return "Search Notion and create pages" }

func (n *Notion) Instructions() string {
	return `Connect an internal Notion integration to search shared content and create pages.

1. Create an internal integration in [Notion integrations](https://www.notion.so/my-integrations).
2. Enable **Read content** and **Insert content**.
3. Open each parent page or database in Notion. Use **Connections** to share it with the integration.
4. Copy the internal integration secret and enter it below.

The connection can only access content shared with it. A successful connection does not grant access to every page.

For database entries, use a **data source ID**, not the database ID.
In Notion, open the database settings, select **Manage data sources**, and copy the data source ID.
`
}

func (n *Notion) Configuration() []configuration.Field {
	return []configuration.Field{{
		Name: "apiToken", Label: "Internal Integration Secret", Type: configuration.FieldTypeString,
		Required: true, Sensitive: true, Description: "Internal integration secret from Notion",
	}}
}

func (n *Notion) Actions() []core.Action   { return []core.Action{&Search{}, &CreatePage{}} }
func (n *Notion) Triggers() []core.Trigger { return nil }

func (n *Notion) Sync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := client.Verify(); err != nil {
		return fmt.Errorf("could not connect to Notion: %w", err)
	}
	ctx.Integration.Ready()
	return nil
}

func (n *Notion) Cleanup(core.IntegrationCleanupContext) error { return nil }
func (n *Notion) Hooks() []core.Hook                           { return nil }
func (n *Notion) HandleHook(core.IntegrationHookContext) error { return nil }
func (n *Notion) HandleRequest(core.HTTPRequestContext)        {}
func (n *Notion) ListResources(string, core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return nil, nil
}
