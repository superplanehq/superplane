package components

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
)

type ListRepositories struct{}

type Configuration struct {
	Integration string `json:"integration"`
}

func (c *ListRepositories) Name() string {
	return "list-repositories"
}

func (c *ListRepositories) Label() string {
	return "List Repositories"
}

func (c *ListRepositories) Description() string {
	return "List GitHub repositories"
}

func (c *ListRepositories) Icon() string {
	return "github"
}

func (c *ListRepositories) Color() string {
	return "gray"
}

func (c *ListRepositories) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (c *ListRepositories) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "integration",
			Label:    "GitHub integration",
			Type:     components.FieldTypeIntegration,
			Required: true,
			TypeOptions: &components.TypeOptions{
				Integration: &components.IntegrationTypeOptions{
					Type: "github",
				},
			},
		},
	}
}

func (c *ListRepositories) Execute(ctx components.ExecutionContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	integration, err := ctx.IntegrationContext.GetIntegration(config.Integration)
	if err != nil {
		return fmt.Errorf("failed to get integration: %w", err)
	}

	//
	// TODO: here I need the specific integration data, not the generic one.
	//

	projects, err := integration.List("repository")
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: {projects},
	})
}

func (c *ListRepositories) Actions() []components.Action {
	// TODO
	return []components.Action{}
}

func (c *ListRepositories) HandleAction(ctx components.ActionContext) error {
	// TODO
	return fmt.Errorf("not supported yet")
}
