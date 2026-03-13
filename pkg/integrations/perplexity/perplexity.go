package perplexity

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("perplexity", &Perplexity{})
}

type Perplexity struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

var agentPresets = []string{"fast-search", "pro-search", "deep-research", "advanced-deep-research"}

func (p *Perplexity) Name() string {
	return "perplexity"
}

func (p *Perplexity) Label() string {
	return "Perplexity"
}

func (p *Perplexity) Icon() string {
	return "perplexity"
}

func (p *Perplexity) Description() string {
	return "Run AI agents with Perplexity"
}

func (p *Perplexity) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Perplexity API key (pplx-...)",
		},
	}
}

func (p *Perplexity) Components() []core.Component {
	return []core.Component{
		&runAgent{},
	}
}

func (p *Perplexity) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (p *Perplexity) Instructions() string {
	return ""
}

func (p *Perplexity) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (p *Perplexity) Sync(ctx core.SyncContext) error {
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

func (p *Perplexity) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (p *Perplexity) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "agent-preset":
		resources := make([]core.IntegrationResource, 0, len(agentPresets))
		for _, preset := range agentPresets {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: preset,
				ID:   preset,
			})
		}
		return resources, nil

	case "agent-model":
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

	return []core.IntegrationResource{}, nil
}

func (p *Perplexity) Actions() []core.Action {
	return []core.Action{}
}

func (p *Perplexity) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
