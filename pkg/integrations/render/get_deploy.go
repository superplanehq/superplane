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

const GetDeployPayloadType = "render.deploy"

type GetDeploy struct{}

type GetDeployConfiguration struct {
	Service  string `json:"service" mapstructure:"service"`
	DeployID string `json:"deployId" mapstructure:"deployId"`
}

func (c *GetDeploy) Name() string {
	return "render.getDeploy"
}

func (c *GetDeploy) Label() string {
	return "Get Deploy"
}

func (c *GetDeploy) Description() string {
	return "Retrieve a deploy by ID for a Render service"
}

func (c *GetDeploy) Documentation() string {
	return `The Get Deploy component fetches a deploy for a Render service.

## Use Cases

- **Status checks**: Inspect deploy status and timestamps
- **Debugging**: Fetch deploy metadata after receiving an event

## Configuration

- **Service**: Render service that owns the deploy
- **Deploy ID**: Deploy ID to retrieve (supports expressions)

## Output

Emits a ` + "`render.deploy`" + ` payload containing deploy fields like ` + "`deployId`" + `, ` + "`status`" + `, ` + "`trigger`" + `, and timestamps when available.`
}

func (c *GetDeploy) Icon() string {
	return "rocket"
}

func (c *GetDeploy) Color() string {
	return "gray"
}

func (c *GetDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetDeploy) Configuration() []configuration.Field {
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
			Description: "Render deploy ID to retrieve",
		},
	}
}

func decodeGetDeployConfiguration(configuration any) (GetDeployConfiguration, error) {
	spec := GetDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return GetDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	spec.DeployID = strings.TrimSpace(spec.DeployID)
	if spec.Service == "" {
		return GetDeployConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.DeployID == "" {
		return GetDeployConfiguration{}, fmt.Errorf("deployId is required")
	}

	return spec, nil
}

func (c *GetDeploy) Setup(ctx core.SetupContext) error {
	_, err := decodeGetDeployConfiguration(ctx.Configuration)
	return err
}

func (c *GetDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetDeploy) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.GetDeploy(spec.Service, spec.DeployID)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetDeployPayloadType,
		[]any{deployDataFromDeployResponse(spec.Service, deploy)},
	)
}

func (c *GetDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetDeploy) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetDeploy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}
