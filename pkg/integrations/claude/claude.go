package claude

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runagent"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runcloudagent"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("claude", &Claude{})
}

type Claude struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

func (i *Claude) Name() string {
	return "claude"
}

func (i *Claude) Label() string {
	return "Claude"
}

func (i *Claude) Icon() string {
	return "loader"
}

func (i *Claude) Description() string {
	return "Use Claude models in workflows"
}

func (i *Claude) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Claude API key",
			Required:    true,
		},
	}
}

func (i *Claude) Actions() []core.Action {
	return []core.Action{
		&TextPrompt{},
		&runagent.RunAgent{},
		&runcloudagent.RunCloudAgent{},
	}
}

func (i *Claude) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (i *Claude) Instructions() string {
	return "To get new Claude API key, go to [platform.claude.com](https://platform.claude.com)."
}

func (i *Claude) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (i *Claude) Sync(ctx core.SyncContext) error {
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

func (i *Claude) HandleRequest(ctx core.HTTPRequestContext) {
}

func (i *Claude) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "model":
		return i.listModels(resourceType, ctx)
	case "agent":
		return i.listAgents(resourceType, ctx)
	case "environment":
		return i.listEnvironments(resourceType, ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func (i *Claude) listModels(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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

func (i *Claude) listAgents(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := runagent.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	agents, err := client.ListAgents()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(agents))
	for _, agent := range agents {
		if agent.ID == "" {
			continue
		}

		name := agent.Name
		if name == "" {
			name = agent.ID
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   agent.ID,
		})
	}

	return resources, nil
}

func (i *Claude) listEnvironments(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := runagent.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	environments, err := client.ListEnvironments()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(environments))
	for _, environment := range environments {
		if environment.ID == "" {
			continue
		}

		name := environment.Name
		if name == "" {
			name = environment.ID
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   environment.ID,
		})
	}

	return resources, nil
}

func (i *Claude) Hooks() []core.Hook {
	return []core.Hook{}
}

func (i *Claude) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
