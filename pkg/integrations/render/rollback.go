package render

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type Rollback struct{}

type RollbackConfiguration struct {
	Service  string `json:"service" mapstructure:"service"`
	DeployID string `json:"deployId" mapstructure:"deployId"`
}

func (c *Rollback) Name() string {
	return "render.rollback"
}

func (c *Rollback) Label() string {
	return "Rollback"
}

func (c *Rollback) Description() string {
	return "Rollback a Render service to a previous deploy"
}

func (c *Rollback) Documentation() string {
	return `The Rollback component rolls back a Render service to a previously successful deploy.

## Use Cases

- **Incident response**: Quickly revert to the last known-good deploy
- **Automated rollback**: Roll back automatically when a health check fails after deploy
- **Version pinning**: Return to a specific deploy version

## Configuration

- **Service**: The Render service to roll back
- **Deploy ID**: The ID of a previous deploy to roll back to (supports expressions)

## Output

Returns the new deploy object created by the rollback, including:
- Deploy ID and status
- Created and finished timestamps`
}

func (c *Rollback) Icon() string {
	return "rotate-ccw"
}

func (c *Rollback) Color() string {
	return "gray"
}

func (c *Rollback) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *Rollback) Configuration() []configuration.Field {
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
			Description: "The ID of a previous deploy to roll back to",
		},
	}
}

func (c *Rollback) Setup(ctx core.SetupContext) error {
	config, err := decodeRollbackConfiguration(ctx.Configuration)
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

func (c *Rollback) Execute(ctx core.ExecutionContext) error {
	config, err := decodeRollbackConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.Rollback(config.Service, config.DeployID)
	if err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"render.deploy.rollback",
		[]any{map[string]any{
			"deployId":   deploy.ID,
			"status":     deploy.Status,
			"createdAt":  deploy.CreatedAt,
			"finishedAt": deploy.FinishedAt,
		}},
	)
}

func (c *Rollback) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Rollback) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *Rollback) Actions() []core.Action {
	return []core.Action{}
}

func (c *Rollback) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *Rollback) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *Rollback) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeRollbackConfiguration(configuration any) (RollbackConfiguration, error) {
	config := RollbackConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return RollbackConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Service = strings.TrimSpace(config.Service)
	config.DeployID = strings.TrimSpace(config.DeployID)
	return config, nil
}
