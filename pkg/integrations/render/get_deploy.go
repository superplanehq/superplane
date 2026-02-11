package render

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

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
	return "Get the status and details of a Render deploy"
}

func (c *GetDeploy) Documentation() string {
	return `The Get Deploy component retrieves detailed information about a specific deploy for a Render service.

## Use Cases

- **Deploy monitoring**: Check the status of a deploy triggered earlier in the pipeline
- **Status gating**: Wait for a deploy to reach a specific status before proceeding
- **Audit logging**: Retrieve deploy details for tracking and reporting

## Configuration

- **Service**: The Render service that owns the deploy
- **Deploy ID**: The ID of the deploy to retrieve (supports expressions)

## Output

Returns the deploy object including:
- Deploy ID and status (e.g. "live", "build_failed", "canceled")
- Created and finished timestamps`
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
			Description: "The ID of the deploy to retrieve",
		},
	}
}

func (c *GetDeploy) Setup(ctx core.SetupContext) error {
	config, err := decodeGetDeployConfiguration(ctx.Configuration)
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

func (c *GetDeploy) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.GetDeploy(config.Service, config.DeployID)
	if err != nil {
		return fmt.Errorf("failed to get deploy: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"render.deploy",
		[]any{map[string]any{
			"deployId":   deploy.ID,
			"status":     deploy.Status,
			"createdAt":  deploy.CreatedAt,
			"finishedAt": deploy.FinishedAt,
		}},
	)
}

func (c *GetDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
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

func decodeGetDeployConfiguration(configuration any) (GetDeployConfiguration, error) {
	config := GetDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return GetDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Service = strings.TrimSpace(config.Service)
	config.DeployID = strings.TrimSpace(config.DeployID)
	return config, nil
}
