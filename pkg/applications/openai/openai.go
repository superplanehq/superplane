package openai

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterApplication("openai", &OpenAI{})
}

type OpenAI struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
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

func (o *OpenAI) InstallationInstructions() string {
	return ""
}

func (o *OpenAI) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return err
	}

	ctx.AppInstallation.SetState("ready", "")
	return nil
}

func (o *OpenAI) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (o *OpenAI) CompareWebhookConfig(a, b any) (bool, error) {
	return true, nil
}

func (o *OpenAI) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	if resourceType != "model" {
		return []core.ApplicationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return nil, err
	}

	models, err := client.ListModels()
	if err != nil {
		return nil, err
	}

	resources := make([]core.ApplicationResource, 0, len(models))
	for _, model := range models {
		if model.ID == "" {
			continue
		}

		resources = append(resources, core.ApplicationResource{
			Type: resourceType,
			Name: model.ID,
			ID:   model.ID,
		})
	}

	return resources, nil
}

func (o *OpenAI) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (o *OpenAI) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
