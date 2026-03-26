package openrouter

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("openrouter", &OpenRouter{})
}

type OpenRouter struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

func (i *OpenRouter) Name() string {
	return "openrouter"
}

func (i *OpenRouter) Label() string {
	return "OpenRouter"
}

func (i *OpenRouter) Icon() string {
	return "loader"
}

func (i *OpenRouter) Description() string {
	return "Use multiple AI models via OpenRouter's unified API"
}

func (i *OpenRouter) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "OpenRouter API key",
			Required:    true,
		},
	}
}

func (i *OpenRouter) Components() []core.Component {
	return []core.Component{
		&TextPrompt{},
		&GetRemainingCredits{},
		&GetCurrentKeyDetails{},
	}
}

func (i *OpenRouter) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (i *OpenRouter) Instructions() string {
	return "Get your API key at [openrouter.ai/keys](https://openrouter.ai/keys). OpenRouter provides a single API for many models (OpenAI, Anthropic, Google, etc.)."
}

func (i *OpenRouter) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (i *OpenRouter) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
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

func (i *OpenRouter) HandleRequest(ctx core.HTTPRequestContext) {}

func (i *OpenRouter) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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

func (i *OpenRouter) Actions() []core.Action {
	return []core.Action{}
}

func (i *OpenRouter) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
