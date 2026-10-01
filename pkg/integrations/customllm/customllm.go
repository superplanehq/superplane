package customllm

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration(IntegrationName, &CustomLLM{})
}

const IntegrationName = "customLlm"

const (
	APITypeAnthropic        = "anthropic"
	APITypeOpenAI           = "openai"
	APITypeOpenAICompatible = "openai-compatible"
)

type CustomLLM struct{}

type Configuration struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseURL"`
	APIType string `json:"apiType"`
}

func NormalizeAPIType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case APITypeAnthropic, APITypeOpenAI, APITypeOpenAICompatible:
		return strings.ToLower(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("API type must be Anthropic, OpenAI, or OpenAI-compatible")
	}
}

func (c *CustomLLM) Name() string { return IntegrationName }

func (c *CustomLLM) Label() string { return "Custom provider" }

func (c *CustomLLM) Icon() string { return "key" }

func (c *CustomLLM) Description() string {
	return "Use a model provider with your URL, token, and API type"
}

func (c *CustomLLM) Instructions() string {
	return `Set the provider API URL, token, and API type.

- **API URL**: The base URL for the provider, including the version path when the provider uses one.
- **API token**: The token the provider issued for this organization.
- **API type**: Anthropic, OpenAI, or OpenAI-compatible. Match the endpoint the provider documents.`
}

func (c *CustomLLM) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseURL",
			Label:       "API URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Base URL for the model provider",
			Placeholder: "https://example.com/v1",
		},
		{
			Name:        "apiKey",
			Label:       "API token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Token for the model provider",
		},
		{
			Name:        "apiType",
			Label:       "API type",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Anthropic, OpenAI, or OpenAI-compatible",
		},
	}
}

func (c *CustomLLM) Actions() []core.Action { return []core.Action{} }

func (c *CustomLLM) Triggers() []core.Trigger { return []core.Trigger{} }

func (c *CustomLLM) Cleanup(ctx core.IntegrationCleanupContext) error { return nil }

func (c *CustomLLM) Sync(ctx core.SyncContext) error {
	config, err := decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateConfiguration(config); err != nil {
		return err
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if _, err := client.ListModels(); err != nil {
		return err
	}
	ctx.Integration.Ready()
	return nil
}

func (c *CustomLLM) HandleRequest(ctx core.HTTPRequestContext) {}

func (c *CustomLLM) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(model.Name)
		if name == "" {
			name = id
		}
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   id,
		})
	}
	return resources, nil
}

func (c *CustomLLM) Hooks() []core.Hook { return []core.Hook{} }

func (c *CustomLLM) HandleHook(ctx core.IntegrationHookContext) error { return nil }

func decodeConfiguration(raw any) (Configuration, error) {
	config := Configuration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return Configuration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	return config, nil
}

func validateConfiguration(config Configuration) error {
	if strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("apiKey is required")
	}
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		return fmt.Errorf("baseURL is required")
	}
	if err := llm.ValidateBaseURL(baseURL); err != nil {
		return fmt.Errorf("enter a public http or https URL")
	}
	if _, err := NormalizeAPIType(config.APIType); err != nil {
		return err
	}
	return nil
}
