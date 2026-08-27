package vercel

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateProject struct{}

type CreateProjectConfiguration struct {
	Name      string `json:"name" mapstructure:"name"`
	Framework string `json:"framework" mapstructure:"framework"`
}

func (c *CreateProject) Name() string {
	return "vercel.createProject"
}

func (c *CreateProject) Label() string {
	return "Create Project"
}

func (c *CreateProject) Description() string {
	return "Create a new Vercel project"
}

func (c *CreateProject) Documentation() string {
	return `The Create Project component creates a new Vercel project.

## Use Cases

- **Environment provisioning**: Create a project per feature branch or tenant
- **Bootstrap automation**: Set up projects from templates or external systems

## Configuration

- **Name**: Required. The project name. Must be 100 characters or fewer and contain only lowercase letters, numbers, and hyphens.
- **Framework**: Optional. A Vercel framework preset (e.g. ` + "`nextjs`" + `, ` + "`vite`" + `, ` + "`sveltekit`" + `).

## Output

Emits a ` + "`vercel.project`" + ` payload for the created project.`
}

func (c *CreateProject) Icon() string {
	return "folder-plus"
}

func (c *CreateProject) Color() string {
	return "gray"
}

func (c *CreateProject) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateProject) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "my-new-project",
			Description: "Name of the new project",
		},
		{
			Name:        "framework",
			Label:       "Framework",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., nextjs, vite, sveltekit",
			Description: "Optional Vercel framework preset",
		},
	}
}

func decodeCreateProjectConfiguration(configuration any) (CreateProjectConfiguration, error) {
	spec := CreateProjectConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Name = strings.TrimSpace(spec.Name)
	spec.Framework = strings.TrimSpace(strings.ToLower(spec.Framework))

	if spec.Name == "" {
		return spec, fmt.Errorf("name is required")
	}

	return spec, nil
}

func (c *CreateProject) Setup(ctx core.SetupContext) error {
	_, err := decodeCreateProjectConfiguration(ctx.Configuration)
	return err
}

func (c *CreateProject) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCreateProjectConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.CreateProject(spec.Name, spec.Framework)
	if err != nil {
		return fmt.Errorf("failed to create Vercel project: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ProjectPayloadType,
		[]any{projectData(project)},
	)
}

func (c *CreateProject) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateProject) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateProject) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateProject) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateProject) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
