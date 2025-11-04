package components

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
)

type ListProjects struct{}

type Configuration struct {
	Integration string `json:"integration"`
}

func (c *ListProjects) Name() string {
	return "list-projects"
}

func (c *ListProjects) Label() string {
	return "List Projects"
}

func (c *ListProjects) Description() string {
	return "List Semaphore projects"
}

func (c *ListProjects) Icon() string {
	return "workflow"
}

func (c *ListProjects) Color() string {
	return "blue"
}

func (c *ListProjects) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (c *ListProjects) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "integration",
			Label:    "Semaphore integration",
			Type:     components.FieldTypeIntegration,
			Required: true,
			TypeOptions: &components.TypeOptions{
				Integration: &components.IntegrationTypeOptions{
					Type: "semaphore",
				},
			},
		},
	}
}

func (c *ListProjects) Execute(ctx components.ExecutionContext) error {
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

	projects, err := integration.List("project")
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: {projects},
	})
}

func (c *ListProjects) Actions() []components.Action {
	return []components.Action{}
}

func (c *ListProjects) HandleAction(ctx components.ActionContext) error {
	return nil
}
