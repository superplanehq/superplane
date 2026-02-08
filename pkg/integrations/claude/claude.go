package claude

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("claude", &Claude{})
}

type Claude struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

func (i *Claude) Name() string {
	return "claude"
}

func (i *Claude) Label() string {
	return "Claude"
}

func (i *Claude) Icon() string {
	return "loader"
}

func (i *Claude) Description() string {
	return "Use Claude models in workflows"
}

func (i *Claude) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Claude API key",
			Required:    true,
		},
	}
}

func (i *Claude) Components() []core.Component {
	return []core.Component{
		&TextPrompt{},
	}
}

func (i *Claude) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (i *Claude) Instructions() string {
	return "To get new Claude API key, go to [platform.claude.com](https://platform.claude.com)."
}

func (i *Claude) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (i *Claude) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func (i *Claude) HandleRequest(ctx core.HTTPRequestContext) {
}

func (i *Claude) CompareWebhookConfig(a, b any) (bool, error) {
	return true, nil
}

func (i *Claude) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "model" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	models, err := client.ListModels()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(models))
	for _, model := range models {
		if model.ID == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: model.ID,
			ID:   model.ID,
		})
	}

	return resources, nil
}

func (i *Claude) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (i *Claude) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}

func (i *Claude) Actions() []core.Action {
	return []core.Action{}
}

func (i *Claude) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
