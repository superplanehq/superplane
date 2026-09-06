package vercel

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type RollbackProduction struct{}

type RollbackConfiguration struct {
	Project     string `json:"project" mapstructure:"project"`
	Deployment  string `json:"deploymentId" mapstructure:"deploymentId"`
	Description string `json:"description" mapstructure:"description"`
}

func (c *RollbackProduction) Name() string {
	return "vercel.rollback"
}

func (c *RollbackProduction) Label() string {
	return "Rollback Production"
}

func (c *RollbackProduction) Description() string {
	return "Point Vercel production traffic back to a previous production deployment"
}

func (c *RollbackProduction) Documentation() string {
	return `The Rollback Production component points production traffic of a Vercel project to a previous production deployment.

## Use Cases

- **Incident response**: Revert production after a bad release
- **Automated recovery**: Roll back when smoke tests fail after a deploy

## Configuration

- **Project**: Required. The project to roll back.
- **Deployment ID**: Required. A previous **production** deployment to roll back to (starts with ` + "`dpl_`" + `, supports expressions).
- **Description**: Optional. The reason for the rollback, shown in the Vercel dashboard.

## Notes

- Only previous production deployments can be used as rollback targets
- Use the **List Deployments** component to find the deployment to roll back to`
}

func (c *RollbackProduction) Icon() string {
	return "undo-2"
}

func (c *RollbackProduction) Color() string {
	return "gray"
}

func (c *RollbackProduction) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RollbackProduction) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
			Description: "Vercel project to roll back",
		},
		{
			Name:        "deploymentId",
			Label:       "Deployment ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., dpl_... or {{$['Node Name'].data.deploymentId}}",
			Description: "Previous production deployment to roll back to",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional reason for the rollback",
		},
	}
}

func decodeRollbackConfiguration(configuration any) (RollbackConfiguration, error) {
	spec := RollbackConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Project = strings.TrimSpace(spec.Project)
	spec.Deployment = strings.TrimSpace(spec.Deployment)
	spec.Description = strings.TrimSpace(spec.Description)

	if spec.Project == "" {
		return spec, fmt.Errorf("project is required")
	}
	if spec.Deployment == "" {
		return spec, fmt.Errorf("deploymentId is required")
	}

	return spec, nil
}

func (c *RollbackProduction) Setup(ctx core.SetupContext) error {
	_, err := decodeRollbackConfiguration(ctx.Configuration)
	return err
}

func (c *RollbackProduction) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRollbackConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.RollbackProduction(spec.Project, spec.Deployment, spec.Description); err != nil {
		return fmt.Errorf("failed to roll back Vercel production: %w", err)
	}

	data := map[string]any{
		"projectId":    spec.Project,
		"deploymentId": spec.Deployment,
		"rolledBack":   true,
	}
	if spec.Description != "" {
		data["description"] = spec.Description
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		RollbackPayloadType,
		[]any{data},
	)
}

func (c *RollbackProduction) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RollbackProduction) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RollbackProduction) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RollbackProduction) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RollbackProduction) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
