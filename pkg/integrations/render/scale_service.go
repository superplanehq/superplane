package render

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ScaleServicePayloadType = "render.service.scaled"

type ScaleService struct{}

type ScaleServiceConfiguration struct {
	Service      string `json:"service" mapstructure:"service"`
	NumInstances int    `json:"numInstances" mapstructure:"numInstances"`
}

func (c *ScaleService) Name() string {
	return "render.scaleService"
}

func (c *ScaleService) Label() string {
	return "Scale Service"
}

func (c *ScaleService) Description() string {
	return "Scale a Render service to a fixed number of instances"
}

func (c *ScaleService) Documentation() string {
	return `The Scale Service component changes a Render service's manual scaling target.

## Use Cases

- **Traffic preparation**: Increase service capacity before a known launch, load test, or migration
- **Cost cleanup**: Scale non-production services back down after a demo or scheduled test window
- **Multi-service orchestration**: Scale a web service and background worker together from one SuperPlane workflow

## Configuration

- **Service**: Render service to scale
- **Instances**: Fixed instance count to request, from 1 to 100

## Output

Emits a ` + "`render.service.scaled`" + ` payload with ` + "`serviceId`" + `, ` + "`numInstances`" + `, and ` + "`status`" + `.`
}

func (c *ScaleService) Icon() string {
	return "gauge"
}

func (c *ScaleService) Color() string {
	return "gray"
}

func (c *ScaleService) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ScaleService) Configuration() []configuration.Field {
	minInstances := 1
	maxInstances := 100
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
			Description: "Render service to scale",
		},
		{
			Name:        "numInstances",
			Label:       "Instances",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     1,
			Description: "Fixed number of instances to run",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: &minInstances,
					Max: &maxInstances,
				},
			},
		},
	}
}

func decodeScaleServiceConfiguration(configuration any) (ScaleServiceConfiguration, error) {
	spec := ScaleServiceConfiguration{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return ScaleServiceConfiguration{}, fmt.Errorf("failed to create configuration decoder: %w", err)
	}

	if err := decoder.Decode(configuration); err != nil {
		return ScaleServiceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	if spec.Service == "" {
		return ScaleServiceConfiguration{}, errors.New("service is required")
	}

	if spec.NumInstances < 1 || spec.NumInstances > 100 {
		return ScaleServiceConfiguration{}, errors.New("numInstances must be between 1 and 100")
	}

	return spec, nil
}

func (c *ScaleService) Setup(ctx core.SetupContext) error {
	_, err := decodeScaleServiceConfiguration(ctx.Configuration)
	return err
}

func (c *ScaleService) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ScaleService) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeScaleServiceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.ScaleService(spec.Service, spec.NumInstances); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ScaleServicePayloadType,
		[]any{
			map[string]any{
				"serviceId":    spec.Service,
				"numInstances": spec.NumInstances,
				"status":       "accepted",
			},
		},
	)
}

func (c *ScaleService) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ScaleService) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ScaleService) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ScaleService) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ScaleService) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
