package cursor

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("cursor", &Cursor{})
}

type Cursor struct{}

type Configuration struct {
	LaunchAgentKey string `json:"launchAgentKey"`
	AdminKey       string `json:"adminKey"`
}

func (i *Cursor) Name() string {
	return "cursor"
}

func (i *Cursor) Label() string {
	return "Cursor"
}

func (i *Cursor) Icon() string {
	return "cpu"
}

func (i *Cursor) Description() string {
	return "Build workflows with Cursor AI Agents and track usage"
}

func (i *Cursor) Instructions() string {
	return "To get your API keys, visit the [Cursor Dashboard](https://cursor.com/dashboard). You may need separate keys for Agents and Admin features."
}

func (i *Cursor) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "launchAgentKey",
			Label:       "Cloud Agent API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Required for launching AI Agents and downloading agent artifacts.",
			Required:    false,
		},
		{
			Name:        "adminKey",
			Label:       "Admin API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "(For Teams) Required for fetching Usage Data.",
			Required:    false,
		},
	}
}

func (i *Cursor) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.LaunchAgentKey == "" && config.AdminKey == "" {
		return fmt.Errorf("one of the keys is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if config.LaunchAgentKey != "" {
		if err := client.VerifyLaunchAgent(); err != nil {
			return fmt.Errorf("cloud agent key verification failed: %w", err)
		}
	}

	if config.AdminKey != "" {
		if err := client.VerifyAdmin(); err != nil {
			return fmt.Errorf("admin key verification failed: %w", err)
		}
	}

	ctx.Integration.Ready()
	return nil
}

func (i *Cursor) Actions() []core.Action {
	return []core.Action{
		&LaunchAgent{},
		&GetDailyUsageData{},
		&DownloadArtifact{},
	}
}

func (i *Cursor) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (i *Cursor) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (i *Cursor) HandleRequest(ctx core.HTTPRequestContext) {}

func (i *Cursor) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType == "model" {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}

		models, err := client.ListModels()
		if err != nil {
			return nil, err
		}

		resources := []core.IntegrationResource{
			{Type: "model", ID: "", Name: "Auto (Recommended)"},
		}

		for _, model := range models {
			resources = append(resources, core.IntegrationResource{
				Type: "model",
				ID:   model,
				Name: model,
			})
		}

		return resources, nil
	}

	if resourceType == "agent" {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}

		agents, err := client.ListAgents(100)
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(agents))
		for _, agent := range agents {
			name := agent.Name
			if name == "" {
				name = agent.ID
			}

			resources = append(resources, core.IntegrationResource{
				Type: "agent",
				ID:   agent.ID,
				Name: name,
			})
		}

		return resources, nil
	}

	if resourceType == "artifact" {
		agentID := ctx.Parameters["agent"]
		if agentID == "" || strings.Contains(agentID, "{{") {
			return []core.IntegrationResource{}, nil
		}

		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}

		artifacts, err := client.ListArtifacts(agentID)
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(artifacts))
		for _, artifact := range artifacts {
			resources = append(resources, core.IntegrationResource{
				Type: "artifact",
				ID:   artifact.Path,
				Name: artifact.Path,
			})
		}

		return resources, nil
	}

	return []core.IntegrationResource{}, nil
}

func (i *Cursor) Hooks() []core.Hook {
	return []core.Hook{}
}

func (i *Cursor) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
