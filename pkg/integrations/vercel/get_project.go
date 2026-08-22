package vercel

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetProject struct{}

type GetProjectConfiguration struct {
	Project string `json:"project" mapstructure:"project"`
}

func (c *GetProject) Name() string {
	return "vercel.getProject"
}

func (c *GetProject) Label() string {
	return "Get Project"
}

func (c *GetProject) Description() string {
	return "Retrieve a Vercel project by ID or name"
}

func (c *GetProject) Documentation() string {
	return `The Get Project component fetches a Vercel project.

## Use Cases

- **Project details**: Read the framework and settings of a project
- **Guardrails**: Check that a project exists before deploying to it

## Configuration

- **Project**: Required. The Vercel project to retrieve.

## Output

Emits a ` + "`vercel.project`" + ` payload with fields like ` + "`projectId`" + `, ` + "`name`" + `, and ` + "`framework`" + `.`
}

func (c *GetProject) Icon() string {
	return "folder"
}

func (c *GetProject) Color() string {
	return "gray"
}

func (c *GetProject) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetProject) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: "project"}},
			Description: "Vercel project to retrieve",
		},
	}
}

func decodeProjectConfiguration(configuration any) (GetProjectConfiguration, error) {
	spec := GetProjectConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Project = strings.TrimSpace(spec.Project)
	if spec.Project == "" {
		return spec, fmt.Errorf("project is required")
	}

	return spec, nil
}

func (c *GetProject) Setup(ctx core.SetupContext) error {
	_, err := decodeProjectConfiguration(ctx.Configuration)
	return err
}

func (c *GetProject) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeProjectConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(spec.Project)
	if err != nil {
		return fmt.Errorf("failed to fetch Vercel project: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ProjectPayloadType,
		[]any{projectData(project)},
	)
}

func (c *GetProject) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetProject) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetProject) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetProject) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetProject) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
