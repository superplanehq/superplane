package coolify

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeployApplicationPayloadType = "coolify.application.deploy.queued"

type DeployApplication struct{}

type DeployApplicationSpec struct {
	Application string `json:"application" mapstructure:"application"`
	Force       bool   `json:"force" mapstructure:"force"`
}

func (c *DeployApplication) Name() string {
	return "coolify.deployApplication"
}

func (c *DeployApplication) Label() string {
	return "Deploy Application"
}

func (c *DeployApplication) Description() string {
	return "Queue a deployment for a Coolify application (fire-and-forget)"
}

func (c *DeployApplication) Documentation() string {
	return `The Deploy Application component queues a deployment for a Coolify application. It does not wait for the deployment to finish — the execution emits as soon as Coolify acknowledges the request.

## How It Works

1. Calls ` + "`GET /api/v1/deploy?uuid={uuid}&force={force}`" + ` on the configured Coolify instance
2. Emits the queued deployment metadata (deployment UUID, message) on the default output channel and finishes immediately

## Configuration

- **Application**: The Coolify application to deploy
- **Force**: When enabled, Coolify rebuilds the image instead of redeploying the existing one (defaults to false)

## Output

A ` + "`coolify.application.deploy.queued`" + ` payload with ` + "`applicationUuid`" + `, ` + "`deploymentUuid`" + ` (when returned), and ` + "`message`" + `.
`
}

func (c *DeployApplication) Icon() string {
	return "coolify"
}

func (c *DeployApplication) Color() string {
	return "gray"
}

func (c *DeployApplication) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeployApplication) Configuration() []configuration.Field {
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
			Description: "Coolify application to deploy",
		},
		{
			Name:        "force",
			Label:       "Force rebuild",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Force a fresh build instead of redeploying the existing image",
		},
	}
}

func (c *DeployApplication) Setup(ctx core.SetupContext) error {
	_, err := decodeDeployApplicationSpec(ctx.Configuration)
	return err
}

func (c *DeployApplication) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeployApplication) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeDeployApplicationSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	result, err := client.Deploy(spec.Application, spec.Force)
	if err != nil {
		return fmt.Errorf("deploy application: %w", err)
	}

	payload := map[string]any{
		"applicationUuid": spec.Application,
		"force":           spec.Force,
	}
	if result.Message != "" {
		payload["message"] = result.Message
	}
	if len(result.Deployments) > 0 {
		first := result.Deployments[0]
		if first.DeploymentUUID != "" {
			payload["deploymentUuid"] = first.DeploymentUUID
		}
		if first.Message != "" && payload["message"] == nil {
			payload["message"] = first.Message
		}
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeployApplicationPayloadType, []any{payload})
}

func (c *DeployApplication) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeployApplication) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *DeployApplication) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *DeployApplication) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeployApplication) Cleanup(ctx core.SetupContext) error {
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

	return spec, nil
}
