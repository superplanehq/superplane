package restate

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type RemoveDeployment struct{}

type RemoveDeploymentSpec struct {
	DeploymentID string `json:"deploymentId"`
	Force        bool   `json:"force"`
}

func (c *RemoveDeployment) Name() string {
	return "restate.removeDeployment"
}

func (c *RemoveDeployment) Label() string {
	return "Remove Deployment"
}

func (c *RemoveDeployment) Description() string {
	return "Remove a service deployment from the Restate server"
}

func (c *RemoveDeployment) Icon() string {
	return "repeat"
}

func (c *RemoveDeployment) Color() string {
	return "gray"
}

func (c *RemoveDeployment) Documentation() string {
	return `The Remove Deployment component removes a service deployment from the Restate server.

## Use Cases

- **Cleanup**: Remove old deployments after a successful migration
- **Rollback**: Remove a failed deployment to restore previous state
- **Decommissioning**: Remove service deployments that are no longer needed

## Options

- **Force**: Force removal even if the deployment has active invocations.

## Outputs

The component emits an event containing:
- ` + "`deployment_id`" + `: The ID of the removed deployment
- ` + "`force`" + `: Whether force removal was used
- ` + "`status`" + `: "removed"
`
}

func (c *RemoveDeployment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RemoveDeployment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "deploymentId",
			Label:       "Deployment ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "ID of the deployment to remove (starts with dp_)",
			Placeholder: "dp_...",
		},
		{
			Name:        "force",
			Label:       "Force",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Force removal even if the deployment has active invocations",
		},
	}
}

func (c *RemoveDeployment) Setup(ctx core.SetupContext) error {
	spec := RemoveDeploymentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.DeploymentID == "" {
		return errors.New("deploymentId is required")
	}

	return nil
}

func (c *RemoveDeployment) Execute(ctx core.ExecutionContext) error {
	spec := RemoveDeploymentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.RemoveDeployment(spec.DeploymentID, spec.Force)
	if err != nil {
		return fmt.Errorf("failed to remove deployment: %v", err)
	}

	result := map[string]any{
		"deployment_id": spec.DeploymentID,
		"force":         spec.Force,
		"status":        "removed",
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.deployment.removed",
		[]any{result},
	)
}

func (c *RemoveDeployment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RemoveDeployment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RemoveDeployment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RemoveDeployment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RemoveDeployment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RemoveDeployment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
