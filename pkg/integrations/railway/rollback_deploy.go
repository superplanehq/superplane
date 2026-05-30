package railway

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const RollbackDeployPayloadType = "railway.deployment.rollback"

type RollbackDeploy struct{}

type RollbackDeployConfiguration struct {
	DeployID string `json:"deployId" mapstructure:"deployId"`
}

func (c *RollbackDeploy) Name() string {
	return "railway.rollbackDeploy"
}

func (c *RollbackDeploy) Label() string {
	return "Rollback Deploy"
}

func (c *RollbackDeploy) Description() string {
	return "Roll back to a previous Railway deployment"
}

func (c *RollbackDeploy) Documentation() string {
	return `The Rollback Deploy action rolls a Railway service back to a previous deployment.

## Configuration

- **Deploy ID**: The previous Railway deployment to restore.`
}

func (c *RollbackDeploy) Icon() string {
	return "railway"
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
			Name:        "deployId",
			Label:       "Deploy ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., {{$['Get Deployment'].data.deployId}}",
			Description: "Previous Railway deployment ID to restore",
		},
	}
}

func decodeRollbackDeployConfiguration(configuration any) (RollbackDeployConfiguration, error) {
	spec := RollbackDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return RollbackDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.DeployID = strings.TrimSpace(spec.DeployID)
	if spec.DeployID == "" {
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

	if err := client.RollbackDeployment(spec.DeployID); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, RollbackDeployPayloadType, []any{
		map[string]any{
			"deployId":   spec.DeployID,
			"rolledBack": true,
		},
	})
}

func (c *RollbackDeploy) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RollbackDeploy) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *RollbackDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RollbackDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RollbackDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RollbackDeploy) ExampleOutput() map[string]any {
	return map[string]any{
		"deployId":   "ebda9796-09e4-456f-af60-d1a66dee66a0",
		"rolledBack": true,
	}
}
