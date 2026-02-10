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

type RollbackDeploy struct{}

type RollbackDeployConfiguration struct {
	Service            string `json:"service" mapstructure:"service"`
	RollbackToDeployID string `json:"deployId" mapstructure:"deployId"`
}

func (c *RollbackDeploy) Name() string {
	return "render.rollbackDeploy"
}

func (c *RollbackDeploy) Label() string {
	return "Rollback Deploy"
}

func (c *RollbackDeploy) Description() string {
	return "Roll back a Render service to a previous deploy"
}

func (c *RollbackDeploy) Documentation() string {
	return `The Rollback Deploy component triggers a rollback deploy for a Render service.

## Use Cases

- **Automated recovery**: Roll back after detecting errors in a new deploy
- **One-click rollback**: Trigger rollbacks from an incident workflow

## Configuration

- **Service**: Render service to roll back
- **Deploy ID**: The deploy ID to roll back to (supports expressions)

## Output

Emits a ` + "`render.deploy`" + ` payload for the new rollback deploy, and includes the ` + "`rollbackToDeployId`" + ` field for reference.`
}

func (c *RollbackDeploy) Icon() string {
	return "rotate-ccw"
}

func (c *RollbackDeploy) Color() string {
	return "gray"
}

func (c *RollbackDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RollbackDeploy) Configuration() []configuration.Field {
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
			Description: "Render service to roll back",
		},
		{
			Name:        "deployId",
			Label:       "Deploy ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., dep-... or {{$.event.data.deployId}}",
			Description: "Deploy ID to roll back to",
		},
	}
}

func decodeRollbackDeployConfiguration(configuration any) (RollbackDeployConfiguration, error) {
	spec := RollbackDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return RollbackDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	spec.RollbackToDeployID = strings.TrimSpace(spec.RollbackToDeployID)
	if spec.Service == "" {
		return RollbackDeployConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.RollbackToDeployID == "" {
		return RollbackDeployConfiguration{}, fmt.Errorf("deployId is required")
	}

	return spec, nil
}

func (c *RollbackDeploy) Setup(ctx core.SetupContext) error {
	_, err := decodeRollbackDeployConfiguration(ctx.Configuration)
	return err
}

func (c *RollbackDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RollbackDeploy) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRollbackDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.RollbackDeploy(spec.Service, spec.RollbackToDeployID)
	if err != nil {
		return err
	}

	data := deployDataFromDeployResponse(spec.Service, deploy)
	data["rollbackToDeployId"] = spec.RollbackToDeployID

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetDeployPayloadType,
		[]any{data},
	)
}

func (c *RollbackDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *RollbackDeploy) Actions() []core.Action {
	return []core.Action{}
}

func (c *RollbackDeploy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *RollbackDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RollbackDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}
