package claude

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runagent"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runcodeagent"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("claude", &Claude{})
}

type Claude struct{}

type Configuration struct {
	APIKey   string `json:"apiKey"`
	AdminKey string `json:"adminKey"`
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
		{
			Name:        "adminKey",
			Label:       "Admin API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Admin API key, required for fetching usage and cost reports.",
			Required:    false,
		},
	}
}

func (i *Claude) Actions() []core.Action {
	return []core.Action{
		&TextPrompt{},
		&runagent.RunAgent{},
		&runcodeagent.RunCodeAgent{},
		&GetDailyUsage{},
		&CreateBatchMessage{},
	}
}

func (i *Claude) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (i *Claude) Instructions() string {
	return `To get new Claude API key, go to [platform.claude.com](https://platform.claude.com).

## Files & artifacts

The Files API is in beta: SuperPlane enables it per request via the ` + "`anthropic-beta`" + ` header, so no console toggle is needed and a standard API key suffices. Files and sessions are scoped to the API key's workspace. For **Run Managed Agent** artifacts, the agent must save its deliverables under ` + "`/mnt/session/outputs/`" + `.`
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
	case "model", "agent", "environment", "agentVersion":
	default:
		// Unknown resource type: return empty without touching credentials.
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	switch resourceType {
	case "model":
		return i.listModelResources(client)
	case "agent":
		return i.listAgentResources(client)
	case "environment":
		return i.listEnvironmentResources(client)
	case "agentVersion":
		return i.listAgentVersionResources(client, ctx.Parameters["agent"])
	default:
		return []core.IntegrationResource{}, nil
	}
}

func (i *Claude) listModelResources(client *Client) ([]core.IntegrationResource, error) {
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
			Type: "model",
			Name: model.ID,
			ID:   model.ID,
		})
	}

	return resources, nil
}

func (i *Claude) listAgentResources(client *Client) ([]core.IntegrationResource, error) {
	agents, err := client.ListManagedAgents()
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
			Type: "agent",
			Name: name,
			ID:   agent.ID,
		})
	}

	return resources, nil
}

func (i *Claude) listEnvironmentResources(client *Client) ([]core.IntegrationResource, error) {
	environments, err := client.ListManagedEnvironments()
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
			Type: "environment",
			Name: name,
			ID:   environment.ID,
		})
	}

	return resources, nil
}

// listAgentVersionResources lists an agent's versions newest-first, preceded by
// an explicit "Latest" choice. Latest lets the field be returned to its
// unset/latest state after a specific version was pinned, and gives a newly
// created agent a usable option. The value stays the bare version number (or
// "latest"). An empty agent (nothing selected yet) yields no options.
func (i *Claude) listAgentVersionResources(client *Client, agentID string) ([]core.IntegrationResource, error) {
	if strings.TrimSpace(agentID) == "" {
		return []core.IntegrationResource{}, nil
	}

	versions, err := client.ListManagedAgentVersions(agentID)
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(versions)+1)
	resources = append(resources, core.IntegrationResource{
		Type: "agentVersion",
		Name: "Latest",
		ID:   "latest",
	})
	for _, version := range versions {
		value := strconv.Itoa(version.Version)
		resources = append(resources, core.IntegrationResource{
			Type: "agentVersion",
			Name: value,
			ID:   value,
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
