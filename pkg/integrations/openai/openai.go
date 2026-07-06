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
	APIKey   string `json:"apiKey"`
	AdminKey string `json:"adminKey"`
	BaseURL  string `json:"baseURL"`
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
			Name:        "adminKey",
			Label:       "Admin API Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Organization admin API key (sk-admin-...). Required for fetching usage data.",
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

func (o *OpenAI) Actions() []core.Action {
	return []core.Action{
		&CreateResponse{},
		&GetUsage{},
	}
}

func (o *OpenAI) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (o *OpenAI) Instructions() string {
	return `## OpenAI API Key

Create an [OpenAI API key](https://platform.openai.com/api-keys) and copy it.

- Used for model components like Text Prompt.
- For OpenAI-compatible providers (e.g. Azure OpenAI, Ollama, vLLM), set a custom **Base URL** below.

## Admin API Key (optional)

Only required for the **Get Usage Data** component.

Create an [Admin API key](https://platform.openai.com/settings/organization/admin-keys) and copy it (starts with ` + "`sk-admin-`" + `).

- Only **Organization Owners** can create admin keys.
- Admin keys can read organization usage and costs but cannot call model endpoints.

> **Note:** Both keys are shown only once — store them somewhere safe before continuing.`
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

	// The admin key is optional and only used for usage data, so a failed
	// verification must not block model components from becoming ready.
	if config.AdminKey != "" {
		if err := client.VerifyAdmin(); err != nil && ctx.Logger != nil {
			ctx.Logger.Warnf("admin key verification failed: %v", err)
		}
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

func (o *OpenAI) Hooks() []core.Hook {
	return []core.Hook{}
}

func (o *OpenAI) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
