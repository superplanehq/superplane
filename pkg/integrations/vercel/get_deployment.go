package vercel

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetDeployment struct{}

type GetDeploymentConfiguration struct {
	DeploymentID string `json:"deploymentId" mapstructure:"deploymentId"`
}

func (c *GetDeployment) Name() string {
	return "vercel.getDeployment"
}

func (c *GetDeployment) Label() string {
	return "Get Deployment"
}

func (c *GetDeployment) Description() string {
	return "Retrieve a Vercel deployment by ID"
}

func (c *GetDeployment) Documentation() string {
	return `The Get Deployment component fetches a Vercel deployment by ID.

## Use Cases

- **Status checks**: Inspect the current build state of a deployment
- **Debugging**: Fetch deployment details after receiving an event

## Configuration

- **Deployment ID**: Deployment ID to retrieve (starts with ` + "`dpl_`" + `, supports expressions)

## Output

Emits a ` + "`vercel.deployment`" + ` payload containing fields like ` + "`deploymentId`" + `, ` + "`url`" + `, ` + "`readyState`" + `, ` + "`target`" + `, and ` + "`projectId`" + `.`
}

func (c *GetDeployment) Icon() string {
	return "rocket"
}

func (c *GetDeployment) Color() string {
	return "gray"
}

func (c *GetDeployment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetDeployment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "deploymentId",
			Label:       "Deployment ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., dpl_... or {{$['Node Name'].data.deploymentId}}",
			Description: "Vercel deployment ID to retrieve",
		},
	}
}

func decodeGetDeploymentConfiguration(configuration any) (GetDeploymentConfiguration, error) {
	spec := GetDeploymentConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.DeploymentID = strings.TrimSpace(spec.DeploymentID)
	if spec.DeploymentID == "" {
		return spec, fmt.Errorf("deploymentId is required")
	}

	return spec, nil
}

func (c *GetDeployment) Setup(ctx core.SetupContext) error {
	_, err := decodeGetDeploymentConfiguration(ctx.Configuration)
	return err
}

func (c *GetDeployment) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetDeploymentConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deployment, err := client.GetDeployment(spec.DeploymentID)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetDeploymentPayloadType,
		[]any{deploymentData(deployment)},
	)
}

func (c *GetDeployment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetDeployment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetDeployment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetDeployment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetDeployment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
