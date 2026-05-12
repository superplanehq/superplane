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

func (o *OpenRouter) Name() string {
	return "openrouter"
}

func (o *OpenRouter) Label() string {
	return "OpenRouter"
}

func (o *OpenRouter) Icon() string {
	return "globe"
}

func (o *OpenRouter) Description() string {
	return "Access multiple LLM providers through a single OpenAI-compatible API"
}

func (o *OpenRouter) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "OpenRouter API key",
		},
	}
}

func (o *OpenRouter) Components() []core.Component {
	return []core.Component{
		&TextPrompt{},
	}
}

func (o *OpenRouter) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (o *OpenRouter) Instructions() string {
	return "To get your OpenRouter API key, sign up at [openrouter.ai](https://openrouter.ai)."
}

func (o *OpenRouter) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *OpenRouter) Sync(ctx core.SyncContext) error {
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

func (o *OpenRouter) HandleRequest(ctx core.HTTPRequestContext) {
}

func (o *OpenRouter) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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

func (o *OpenRouter) Actions() []core.Action {
	return []core.Action{}
}

func (o *OpenRouter) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
