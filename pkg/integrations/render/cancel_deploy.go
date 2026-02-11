package render

import (
	"fmt"
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

- **Emergency rollback**: Cancel a bad deploy before it goes live
- **Pipeline abort**: Cancel a deploy when an upstream check fails
- **Manual intervention**: Allow operators to cancel deploys through a workflow

## Configuration

- **Service**: The Render service that owns the deploy
- **Deploy ID**: The ID of the in-progress deploy to cancel (supports expressions)

## Output

Returns the updated deploy object including:
- Deploy ID and updated status
- Created and finished timestamps`
}

func (c *CancelDeploy) Icon() string {
	return "x-circle"
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
			Description: "The ID of the deploy to cancel",
		},
	}
}

func (c *CancelDeploy) Setup(ctx core.SetupContext) error {
	config, err := decodeCancelDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if config.Service == "" {
		return fmt.Errorf("service is required")
	}

	if config.DeployID == "" {
		return fmt.Errorf("deploy ID is required")
	}

	return nil
}

func (c *CancelDeploy) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCancelDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.CancelDeploy(config.Service, config.DeployID)
	if err != nil {
		return fmt.Errorf("failed to cancel deploy: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"render.deploy.canceled",
		[]any{map[string]any{
			"deployId":   deploy.ID,
			"status":     deploy.Status,
			"createdAt":  deploy.CreatedAt,
			"finishedAt": deploy.FinishedAt,
		}},
	)
}

func (c *CancelDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CancelDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
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

func decodeCancelDeployConfiguration(configuration any) (CancelDeployConfiguration, error) {
	config := CancelDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return CancelDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Service = strings.TrimSpace(config.Service)
	config.DeployID = strings.TrimSpace(config.DeployID)
	return config, nil
}
