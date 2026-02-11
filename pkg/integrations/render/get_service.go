package render

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetService struct{}

type GetServiceConfiguration struct {
	Service string `json:"service" mapstructure:"service"`
}

func (c *GetService) Name() string {
	return "render.getService"
}

func (c *GetService) Label() string {
	return "Get Service"
}

func (c *GetService) Description() string {
	return "Get details of a Render service"
}

func (c *GetService) Documentation() string {
	return `The Get Service component retrieves detailed information about a Render service.

## Use Cases

- **Status checking**: Check if a service is suspended before triggering a deploy
- **Service lookup**: Fetch service details for downstream processing
- **Workflow gating**: Use service state to decide whether to proceed with a pipeline

## Configuration

- **Service**: The Render service to retrieve details for

## Output

Returns the full service object including:
- Service ID, name, and type
- Suspension status (string: "not_suspended", "suspended", etc.)
- Auto-deploy setting, repo, and branch
- Created and updated timestamps`
}

func (c *GetService) Icon() string {
	return "server"
}

func (c *GetService) Color() string {
	return "gray"
}

func (c *GetService) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetService) Configuration() []configuration.Field {
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
			Description: "Render service to retrieve",
		},
	}
}

func (c *GetService) Setup(ctx core.SetupContext) error {
	config, err := decodeGetServiceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if config.Service == "" {
		return fmt.Errorf("service is required")
	}

	return nil
}

func (c *GetService) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetServiceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	service, err := client.GetService(config.Service)
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"render.service",
		[]any{map[string]any{
			"id":         service.ID,
			"name":       service.Name,
			"type":       service.Type,
			"suspended":  service.Suspended,
			"autoDeploy": service.AutoDeploy,
			"repo":       service.Repo,
			"branch":     service.Branch,
			"createdAt":  service.CreatedAt,
			"updatedAt":  service.UpdatedAt,
		}},
	)
}

func (c *GetService) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetService) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetService) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetService) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetService) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetService) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeGetServiceConfiguration(configuration any) (GetServiceConfiguration, error) {
	config := GetServiceConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return GetServiceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Service = strings.TrimSpace(config.Service)
	return config, nil
}
