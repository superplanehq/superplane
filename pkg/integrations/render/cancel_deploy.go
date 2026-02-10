package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CancelDeploy struct{}

type CancelDeployConfiguration struct {
	Service  string `json:"service" mapstructure:"service"`
	DeployID string `json:"deployId" mapstructure:"deployId"`
}

func (c *CancelDeploy) Name() string {
	return "render.cancelDeploy"
}

func (c *CancelDeploy) Label() string {
	return "Cancel Deploy"
}

func (c *CancelDeploy) Description() string {
	return "Cancel an in-progress deploy for a Render service"
}

func (c *CancelDeploy) Documentation() string {
	return `The Cancel Deploy component cancels an in-progress deploy for a Render service.

## Use Cases

- **Automated rollback/abort**: Cancel deploys when health checks fail
- **Manual intervention**: Stop a deploy triggered earlier in a workflow

## Configuration

- **Service**: Render service that owns the deploy
- **Deploy ID**: Deploy ID to cancel (supports expressions)

## Output

Emits a ` + "`render.deploy`" + ` payload for the cancelled deploy (status is typically ` + "`canceled`" + `).`
}

func (c *CancelDeploy) Icon() string {
	return "circle-slash-2"
}

func (c *CancelDeploy) Color() string {
	return "gray"
}

func (c *CancelDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CancelDeploy) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "service",
			Label:    "Service",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
			Description: "Render service that owns the deploy",
		},
		{
			Name:        "deployId",
			Label:       "Deploy ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., dep-... or {{$.event.data.deployId}}",
			Description: "Render deploy ID to cancel",
		},
	}
}

func decodeCancelDeployConfiguration(configuration any) (CancelDeployConfiguration, error) {
	spec := CancelDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return CancelDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	spec.DeployID = strings.TrimSpace(spec.DeployID)
	if spec.Service == "" {
		return CancelDeployConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.DeployID == "" {
		return CancelDeployConfiguration{}, fmt.Errorf("deployId is required")
	}

	return spec, nil
}

func (c *CancelDeploy) Setup(ctx core.SetupContext) error {
	_, err := decodeCancelDeployConfiguration(ctx.Configuration)
	return err
}

func (c *CancelDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CancelDeploy) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCancelDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.CancelDeploy(spec.Service, spec.DeployID)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetDeployPayloadType,
		[]any{deployDataFromDeployResponse(spec.Service, deploy)},
	)
}

func (c *CancelDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CancelDeploy) Actions() []core.Action {
	return []core.Action{}
}

func (c *CancelDeploy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CancelDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CancelDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}
