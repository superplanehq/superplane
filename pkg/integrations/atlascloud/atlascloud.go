package atlascloud

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("atlascloud", &AtlasCloud{})
}

type AtlasCloud struct{}

type Configuration struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseURL"`
}

func (a *AtlasCloud) Name() string {
	return "atlascloud"
}

func (a *AtlasCloud) Label() string {
	return "Atlas Cloud"
}

func (a *AtlasCloud) Icon() string {
	return "sparkles"
}

func (a *AtlasCloud) Description() string {
	return "Generate text responses with Atlas Cloud models"
}

func (a *AtlasCloud) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Atlas Cloud API key",
		},
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Custom API base URL for the Atlas Cloud OpenAI-compatible API",
			Placeholder: "https://api.atlascloud.ai/v1",
		},
	}
}

func (a *AtlasCloud) Actions() []core.Action {
	return []core.Action{
		&CreateResponse{},
	}
}

func (a *AtlasCloud) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (a *AtlasCloud) Instructions() string {
	return "To get an Atlas Cloud API key, go to [atlascloud.ai](https://www.atlascloud.ai/?utm_source=github&utm_medium=link&utm_campaign=superplane)."
}

func (a *AtlasCloud) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (a *AtlasCloud) Sync(ctx core.SyncContext) error {
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

func (a *AtlasCloud) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (a *AtlasCloud) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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

func (a *AtlasCloud) Hooks() []core.Hook {
	return []core.Hook{}
}

func (a *AtlasCloud) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
