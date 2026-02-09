package cursor

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("cursor", &Cursor{}, &CursorWebhookHandler{})
}

type Cursor struct{}

type Configuration struct {
	CloudAgentsAPIKey string `json:"cloudAgentsApiKey" mapstructure:"cloudAgentsApiKey"`
	AdminAPIKey       string `json:"adminApiKey" mapstructure:"adminApiKey"`
}

type IntegrationMetadata struct {
	// WebhooksBaseURL is persisted so components can build webhook URLs during execution.
	// ExecutionContext does not include WebhooksBaseURL.
	WebhooksBaseURL string `json:"webhooksBaseURL" mapstructure:"webhooksBaseURL"`
}

// WebhookConfiguration is intentionally empty: Cursor webhooks are configured per-agent,
// but SuperPlane still needs a webhook record for the node so it has a stable endpoint.
type WebhookConfiguration struct{}

func (c *Cursor) Name() string {
	return "cursor"
}

func (c *Cursor) Label() string {
	return "Cursor"
}

func (c *Cursor) Icon() string {
	return "bot"
}

func (c *Cursor) Description() string {
	return "Launch Cursor background agents and fetch team usage analytics"
}

func (c *Cursor) Instructions() string {
	return `Create a Cursor API key in Cursor Dashboard -> Integrations.`
}

func (c *Cursor) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "cloudAgentsApiKey",
			Label:       "Cloud Agent API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Required for launching AI Agents. Found in Cursor Dashboard > Integrations.",
		},
		{
			Name:        "adminApiKey",
			Label:       "Admin API Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Required for fetching Usage Data. Found in Settings > Advanced > Admin API keys.",
		},
	}
}

func (c *Cursor) Components() []core.Component {
	return []core.Component{
		&LaunchCloudAgent{},
		&GetDailyUsageData{},
	}
}

func (c *Cursor) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (c *Cursor) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *Cursor) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.CloudAgentsAPIKey == "" {
		return fmt.Errorf("cloudAgentsApiKey is required")
	}

	cloudClient, err := NewCloudAgentsClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := cloudClient.Verify(); err != nil {
		return err
	}

	// Admin API key is optional; verify when present.
	if config.AdminAPIKey != "" {
		adminClient, err := NewAdminClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return err
		}

		if err := adminClient.Verify(); err != nil {
			return err
		}
	}

	ctx.Integration.SetMetadata(IntegrationMetadata{
		WebhooksBaseURL: ctx.WebhooksBaseURL,
	})

	ctx.Integration.Ready()
	return nil
}

func (c *Cursor) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (c *Cursor) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "model" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewCloudAgentsClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	models, err := client.ListModels()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(models))
	for _, model := range models {
		if model == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: model,
			ID:   model,
		})
	}

	return resources, nil
}

func (c *Cursor) Actions() []core.Action {
	return []core.Action{}
}

func (c *Cursor) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
