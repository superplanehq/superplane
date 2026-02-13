package openai

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("openai", &OpenAI{})
}

type OpenAI struct{}

type Configuration struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseURL"`
}

func (o *OpenAI) Name() string {
	return "openai"
}

func (o *OpenAI) Label() string {
	return "OpenAI"
}

func (o *OpenAI) Icon() string {
	return "sparkles"
}

func (o *OpenAI) Description() string {
	return "Generate text responses with OpenAI models"
}

func (o *OpenAI) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "OpenAI API key",
		},
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Custom API base URL for OpenAI-compatible providers (e.g. Azure OpenAI, Ollama, vLLM)",
			Placeholder: "https://api.openai.com/v1",
		},
	}
}

func (o *OpenAI) Components() []core.Component {
	return []core.Component{
		&CreateResponse{},
	}
}

func (o *OpenAI) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (o *OpenAI) Instructions() string {
	return ""
}

func (o *OpenAI) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *OpenAI) Sync(ctx core.SyncContext) error {
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

func (o *OpenAI) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (o *OpenAI) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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

func (o *OpenAI) Actions() []core.Action {
	return []core.Action{}
}

func (o *OpenAI) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
