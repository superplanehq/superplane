package dokploy

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeployApplicationPayloadType = "dokploy.application.deploy.queued"

type DeployApplication struct{}

type DeployApplicationSpec struct {
	Application string `json:"application" mapstructure:"application"`
	Title       string `json:"title" mapstructure:"title"`
	Description string `json:"description" mapstructure:"description"`
}

func (d *DeployApplication) Name() string {
	return "dokploy.deployApplication"
}

func (d *DeployApplication) Label() string {
	return "Deploy Application"
}

func (d *DeployApplication) Description() string {
	return "Queue a deployment for a Dokploy application (fire-and-forget)"
}

func (d *DeployApplication) Documentation() string {
	return `The Deploy Application component queues a deployment for a Dokploy application. It does not wait for the deployment to finish. The execution emits as soon as Dokploy accepts the request.

## How It Works

1. Calls ` + "`POST /api/application.deploy`" + ` on the configured Dokploy instance with the selected application
2. Emits the queued deployment metadata on the default output channel and finishes immediately

Dokploy adds the deployment to a queue and returns no deployment identifier, so there is nothing to poll for here. Watch the deployment in Dokploy, or gate later steps on your application's own health check.

## Configuration

- **Application**: The Dokploy application to deploy
- **Title**: Optional label recorded against the deployment in Dokploy's log. Dokploy records "Manual deployment" when this is empty.
- **Description**: Optional longer note recorded alongside the title

## Output

A ` + "`dokploy.application.deploy.queued`" + ` payload with ` + "`applicationId`" + `, and ` + "`title`" + ` / ` + "`description`" + ` when set.
`
}

func (d *DeployApplication) Icon() string {
	return "dokploy"
}

func (d *DeployApplication) Color() string {
	return "gray"
}

func (d *DeployApplication) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeployApplication) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "application",
			Label:       "Application",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select application",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeApplication,
				},
			},
			Description: "Dokploy application to deploy",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Manual deployment",
			Description: "Label recorded against this deployment in Dokploy",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Longer note recorded alongside the title",
		},
	}
}

func (d *DeployApplication) Setup(ctx core.SetupContext) error {
	_, err := decodeDeployApplicationSpec(ctx.Configuration)
	return err
}

func (d *DeployApplication) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeployApplication) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeDeployApplicationSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	err = client.Deploy(DeployRequest{
		ApplicationID: spec.Application,
		Title:         spec.Title,
		Description:   spec.Description,
	})
	if err != nil {
		return fmt.Errorf("deploy application: %w", err)
	}

	payload := map[string]any{
		"applicationId": spec.Application,
	}
	if spec.Title != "" {
		payload["title"] = spec.Title
	}
	if spec.Description != "" {
		payload["description"] = spec.Description
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeployApplicationPayloadType, []any{payload})
}

func (d *DeployApplication) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeployApplication) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (d *DeployApplication) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (d *DeployApplication) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeployApplication) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeDeployApplicationSpec(cfg any) (DeployApplicationSpec, error) {
	spec := DeployApplicationSpec{}
	if err := mapstructure.Decode(cfg, &spec); err != nil {
		return DeployApplicationSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Application = strings.TrimSpace(spec.Application)
	if spec.Application == "" {
		return DeployApplicationSpec{}, fmt.Errorf("application is required")
	}

	spec.Title = strings.TrimSpace(spec.Title)
	spec.Description = strings.TrimSpace(spec.Description)
	return spec, nil
}
