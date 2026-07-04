package coolify

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ControlServicePayloadType = "coolify.service.controlled"

type ControlService struct{}

type ControlServiceSpec struct {
	Service   string `json:"service" mapstructure:"service"`
	Operation string `json:"operation" mapstructure:"operation"`
}

func (c *ControlService) Name() string {
	return "coolify.controlService"
}

func (c *ControlService) Label() string {
	return "Control Service"
}

func (c *ControlService) Description() string {
	return "Start, stop, or restart a Coolify service"
}

func (c *ControlService) Documentation() string {
	return `The Control Service component invokes a lifecycle operation on a Coolify service (one-click or custom Docker Compose stack).

## How It Works

1. Calls ` + "`GET /api/v1/services/{uuid}/{operation}`" + ` on the configured Coolify instance
2. Emits the API confirmation message on the default output channel

## Configuration

- **Service**: The Coolify service to control
- **Operation**: One of ` + "`start`" + `, ` + "`stop`" + `, or ` + "`restart`" + `
`
}

func (c *ControlService) Icon() string {
	return "coolify"
}

func (c *ControlService) Color() string {
	return "gray"
}

func (c *ControlService) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ControlService) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select service",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeService,
				},
			},
			Description: "Coolify service to control",
		},
		lifecycleOperationField(),
	}
}

func (c *ControlService) Setup(ctx core.SetupContext) error {
	_, err := decodeControlServiceSpec(ctx.Configuration)
	return err
}

func (c *ControlService) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ControlService) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeControlServiceSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	op := LifecycleOperation(spec.Operation)
	result, err := client.ControlService(spec.Service, op)
	if err != nil {
		return fmt.Errorf("%s service: %w", op, err)
	}

	payload := map[string]any{
		"serviceUuid": spec.Service,
		"operation":   string(op),
		"message":     result.Message,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, ControlServicePayloadType, []any{payload})
}

func (c *ControlService) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ControlService) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *ControlService) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *ControlService) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ControlService) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeControlServiceSpec(cfg any) (ControlServiceSpec, error) {
	spec := ControlServiceSpec{}
	if err := mapstructure.Decode(cfg, &spec); err != nil {
		return ControlServiceSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	if spec.Service == "" {
		return ControlServiceSpec{}, fmt.Errorf("service is required")
	}

	spec.Operation = strings.TrimSpace(spec.Operation)
	if err := validateLifecycleOperation(spec.Operation); err != nil {
		return ControlServiceSpec{}, err
	}

	return spec, nil
}
