package vercel

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CancelDeployment struct{}

type CancelDeploymentConfiguration struct {
	DeploymentID string `json:"deploymentId" mapstructure:"deploymentId"`
}

func (c *CancelDeployment) Name() string {
	return "vercel.cancelDeployment"
}

func (c *CancelDeployment) Label() string {
	return "Cancel Deployment"
}

func (c *CancelDeployment) Description() string {
	return "Cancel an in-progress Vercel deployment"
}

func (c *CancelDeployment) Documentation() string {
	return `The Cancel Deployment component stops a Vercel build that is still in progress.

## Use Cases

- **Wrong-branch recovery**: Stop a build started from the wrong branch
- **Incident response**: Cancel deploys with known errors before they go live
- **Pipeline guardrails**: Cancel queued builds superseded by newer commits

## Configuration

- **Deployment ID**: Required. The deployment to cancel (starts with ` + "`dpl_`" + `, supports expressions).

## Notes

- Only deployments that are still building can be canceled
- The action is irreversible: the build is stopped and marked ` + "`CANCELED`"
}

func (c *CancelDeployment) Icon() string {
	return "x-circle"
}

func (c *CancelDeployment) Color() string {
	return "gray"
}

func (c *CancelDeployment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CancelDeployment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "deploymentId",
			Label:       "Deployment ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., dpl_... or {{$['Node Name'].data.deploymentId}}",
			Description: "Vercel deployment to cancel",
		},
	}
}

func decodeCancelDeploymentConfiguration(configuration any) (CancelDeploymentConfiguration, error) {
	spec := CancelDeploymentConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.DeploymentID = strings.TrimSpace(spec.DeploymentID)
	if spec.DeploymentID == "" {
		return spec, fmt.Errorf("deploymentId is required")
	}

	return spec, nil
}

func (c *CancelDeployment) Setup(ctx core.SetupContext) error {
	_, err := decodeCancelDeploymentConfiguration(ctx.Configuration)
	return err
}

func (c *CancelDeployment) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCancelDeploymentConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deployment, err := client.CancelDeployment(spec.DeploymentID)
	if err != nil {
		return fmt.Errorf("failed to cancel Vercel deployment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetDeploymentPayloadType,
		[]any{deploymentData(deployment)},
	)
}

func (c *CancelDeployment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CancelDeployment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CancelDeployment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CancelDeployment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CancelDeployment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
