package flyio

import (
	"encoding/json"
	"fmt"
	"net/http"

	_ "embed"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListApps struct{}

type ListAppsSpec struct {
	OrgSlug string `json:"orgSlug" mapstructure:"orgSlug"`
}

func (c *ListApps) Name() string {
	return "flyio.listApps"
}

func (c *ListApps) Label() string {
	return "List Apps"
}

func (c *ListApps) Description() string {
	return "List all Apps in a Fly.io organization"
}

func (c *ListApps) Documentation() string {
	return `## List Fly.io Apps

This component retrieves a list of all Apps in your Fly.io organization.

### Use Cases
- Get an overview of all applications in your organization
- Find App names for other operations (create machines, deploy, etc.)
- Inventory and audit your Fly.io resources

### Configuration
- **Organization Slug**: The organization to list Apps from (defaults to integration's configured org)

### Output
The component outputs an array of App details including name, status, and machine count.`
}

func (c *ListApps) Icon() string {
	return "layout-grid"
}

func (c *ListApps) Color() string {
	return "purple"
}

//go:embed list_apps_example_output.json
var listAppsExampleOutput []byte

func (c *ListApps) ExampleOutput() map[string]any {
	var output map[string]any
	// The component rules state that Example JSON files must be valid.
	// We can trust it is valid JSON if we wrote it correctly.
	// Ignoring error here as this is a static asset.
	_ = json.Unmarshal(listAppsExampleOutput, &output)
	return output
}

func (c *ListApps) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  core.DefaultOutputChannel.Name,
			Label: core.DefaultOutputChannel.Label,
		},
	}
}

func (c *ListApps) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "orgSlug",
			Label:       "Organization Slug",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Organization slug to list Apps from (defaults to integration's configured org)",
		},
	}
}

func (c *ListApps) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *ListApps) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListApps) Execute(ctx core.ExecutionContext) error {
	spec := ListAppsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	// Get org slug from component config or fall back to integration config
	orgSlug := spec.OrgSlug
	if orgSlug == "" {
		// Try to get from integration config
		if configSlug, err := ctx.Integration.GetConfig("orgSlug"); err == nil && len(configSlug) > 0 {
			orgSlug = string(configSlug)
		}
	}

	if orgSlug == "" {
		// Try to get from integration metadata
		integrationMetadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err == nil && len(integrationMetadata.Apps) > 0 {
			// Use the org from the first app if available
			if integrationMetadata.Apps[0].Organization != nil {
				orgSlug = integrationMetadata.Apps[0].Organization.Slug
			}
		}
	}

	if orgSlug == "" {
		orgSlug = "personal"
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	apps, err := client.ListApps(orgSlug)
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}

	// Convert to output format
	appInfos := make([]map[string]any, 0, len(apps))
	for _, app := range apps {
		appInfos = append(appInfos, map[string]any{
			"name":         app.Name,
			"id":           app.ID,
			"status":       app.Status,
			"machineCount": app.MachineCount,
			"volumeCount":  app.VolumeCount,
			"network":      app.Network,
		})
	}

	output := map[string]any{
		"orgSlug": orgSlug,
		"apps":    appInfos,
		"count":   len(appInfos),
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"flyio.appList",
		[]any{output},
	)
}

func (c *ListApps) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListApps) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ListApps) Actions() []core.Action {
	return []core.Action{}
}

func (c *ListApps) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ListApps) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
