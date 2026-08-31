package groq

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("groq", &Groq{})
}

type Groq struct{}

type Configuration struct {
	APIKey string `json:"apiKey" mapstructure:"apiKey"`
}

func (g *Groq) Name() string {
	return "groq"
}

func (g *Groq) Label() string {
	return "Groq"
}

func (g *Groq) Icon() string {
	return "groq"
}

func (g *Groq) Description() string {
	return "Generate text responses with models hosted by Groq"
}

func (g *Groq) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Groq API key",
		},
	}
}

func (g *Groq) Actions() []core.Action {
	return []core.Action{&ChatCompletion{}}
}

func (g *Groq) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (g *Groq) Instructions() string {
	return `## Groq API Key

Create a [Groq API key](https://console.groq.com/keys) and copy it.`
}

func (g *Groq) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (g *Groq) Sync(ctx core.SyncContext) error {
	if _, err := configurationAPIKey(ctx.Configuration); err != nil {
		return err
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

func (g *Groq) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (g *Groq) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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
		if !model.IsSelectable() {
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

func (g *Groq) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *Groq) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
