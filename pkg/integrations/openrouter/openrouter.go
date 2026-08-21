package openrouter

import (
	"context"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("openrouter", &OpenRouter{})
}

type OpenRouter struct{}

type Configuration struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseURL"`
}

func (o *OpenRouter) Name() string {
	return "openrouter"
}

func (o *OpenRouter) Label() string {
	return "OpenRouter"
}

func (o *OpenRouter) Icon() string {
	return "openrouter"
}

func (o *OpenRouter) Description() string {
	return "Use OpenRouter models in workflows"
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
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Custom OpenRouter API base URL",
			Placeholder: "https://openrouter.ai/api/v1",
		},
	}
}

func (o *OpenRouter) Actions() []core.Action {
	return []core.Action{}
}

func (o *OpenRouter) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (o *OpenRouter) Instructions() string {
	return `## OpenRouter API Key

Create an [OpenRouter API key](https://openrouter.ai/keys) and copy it.

- Used for OpenRouter agent runs and model components.
- If you use a custom OpenRouter-compatible endpoint, set **Base URL** below.

> **Note:** The key is shown only once. Store it somewhere safe before you continue.`
}

func (o *OpenRouter) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *OpenRouter) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := llm.New(ctx.HTTP, llm.ProviderOpenRouter, llm.Credentials{
		APIKey:  strings.TrimSpace(config.APIKey),
		BaseURL: strings.TrimSpace(config.BaseURL),
	})
	if err != nil {
		return err
	}

	if _, err := client.ListModels(context.Background()); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func (o *OpenRouter) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (o *OpenRouter) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "model" {
		return []core.IntegrationResource{}, nil
	}

	creds, err := credentialsFromIntegration(ctx.Integration)
	if err != nil {
		return nil, err
	}

	client, err := llm.New(ctx.HTTP, llm.ProviderOpenRouter, creds)
	if err != nil {
		return nil, err
	}

	models, err := client.ListModels(context.Background())
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

func (o *OpenRouter) Hooks() []core.Hook {
	return []core.Hook{}
}

func (o *OpenRouter) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}

func credentialsFromIntegration(integration core.IntegrationContext) (llm.Credentials, error) {
	apiKey, err := integration.GetConfig("apiKey")
	if err != nil {
		return llm.Credentials{}, fmt.Errorf("failed to get API key: %w", err)
	}

	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return llm.Credentials{}, fmt.Errorf("apiKey is required")
	}

	creds := llm.Credentials{APIKey: key}
	if baseURL, err := integration.GetConfig("baseURL"); err == nil {
		creds.BaseURL = strings.TrimSpace(string(baseURL))
	}
	return creds, nil
}
