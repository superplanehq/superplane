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

const GetServicePayloadType = "render.service"

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
	return "Retrieve a Render service by ID"
}

func (c *GetService) Documentation() string {
	return `The Get Service component fetches details for a Render service.

## Use Cases

- **Service inspection**: Fetch current service configuration and metadata
- **Workflow context**: Use service fields to drive branching decisions in later steps

## Configuration

- **Service**: Render service to retrieve

## Output

Emits a ` + "`render.service`" + ` payload containing service fields like ` + "`serviceId`" + `, ` + "`serviceName`" + `, ` + "`type`" + `, ` + "`dashboardUrl`" + `, and ` + "`suspended`" + `.`
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

func decodeGetServiceConfiguration(configuration any) (GetServiceConfiguration, error) {
	spec := GetServiceConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return GetServiceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	if spec.Service == "" {
		return GetServiceConfiguration{}, fmt.Errorf("service is required")
	}

	return spec, nil
}

func (c *GetService) Setup(ctx core.SetupContext) error {
	_, err := decodeGetServiceConfiguration(ctx.Configuration)
	return err
}

func (c *GetService) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetService) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetServiceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	service, err := client.GetService(spec.Service)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetServicePayloadType,
		[]any{serviceDataFromService(service)},
	)
}

func (c *GetService) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
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
