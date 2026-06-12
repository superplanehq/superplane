package coolify

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ControlApplicationPayloadType = "coolify.application.controlled"

type ControlApplication struct{}

type ControlApplicationSpec struct {
	Application string `json:"application" mapstructure:"application"`
	Operation   string `json:"operation" mapstructure:"operation"`
}

func (c *ControlApplication) Name() string {
	return "coolify.controlApplication"
}

func (c *ControlApplication) Label() string {
	return "Control Application"
}

func (c *ControlApplication) Description() string {
	return "Start, stop, or restart a Coolify application"
}

func (c *ControlApplication) Documentation() string {
	return `The Control Application component invokes a lifecycle operation on a Coolify application.

## How It Works

1. Calls ` + "`GET /api/v1/applications/{uuid}/{operation}`" + ` on the configured Coolify instance
2. Emits the API confirmation message on the default output channel

## Configuration

- **Application**: The Coolify application to control
- **Operation**: One of ` + "`start`" + `, ` + "`stop`" + `, or ` + "`restart`" + `
`
}

func (c *ControlApplication) Icon() string {
	return "coolify"
}

func (c *ControlApplication) Color() string {
	return "gray"
}

func (c *ControlApplication) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ControlApplication) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "application",
			Label:       "Application",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select application",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeApplication,
				},
			},
			Description: "Coolify application to control",
		},
		lifecycleOperationField(),
	}
}

func (c *ControlApplication) Setup(ctx core.SetupContext) error {
	_, err := decodeControlApplicationSpec(ctx.Configuration)
	return err
}

func (c *ControlApplication) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ControlApplication) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeControlApplicationSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	op := LifecycleOperation(spec.Operation)
	result, err := client.ControlApplication(spec.Application, op)
	if err != nil {
		return fmt.Errorf("%s application: %w", op, err)
	}

	payload := map[string]any{
		"applicationUuid": spec.Application,
		"operation":       string(op),
		"message":         result.Message,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, ControlApplicationPayloadType, []any{payload})
}

func (c *ControlApplication) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ControlApplication) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *ControlApplication) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *ControlApplication) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ControlApplication) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeControlApplicationSpec(cfg any) (ControlApplicationSpec, error) {
	spec := ControlApplicationSpec{}
	if err := mapstructure.Decode(cfg, &spec); err != nil {
		return ControlApplicationSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Application = strings.TrimSpace(spec.Application)
	if spec.Application == "" {
		return ControlApplicationSpec{}, fmt.Errorf("application is required")
	}

	spec.Operation = strings.TrimSpace(spec.Operation)
	if err := validateLifecycleOperation(spec.Operation); err != nil {
		return ControlApplicationSpec{}, err
	}

	return spec, nil
}

// lifecycleOperationField builds the shared lifecycle operation select field
// used by ControlApplication and ControlService.
func lifecycleOperationField() configuration.Field {
	return configuration.Field{
		Name:     "operation",
		Label:    "Operation",
		Type:     configuration.FieldTypeSelect,
		Required: true,
		Default:  string(LifecycleStart),
		TypeOptions: &configuration.TypeOptions{
			Select: &configuration.SelectTypeOptions{
				Options: []configuration.FieldOption{
					{Label: "Start", Value: string(LifecycleStart)},
					{Label: "Stop", Value: string(LifecycleStop)},
					{Label: "Restart", Value: string(LifecycleRestart)},
				},
			},
		},
		Description: "Lifecycle operation to invoke",
	}
}

func validateLifecycleOperation(op string) error {
	op = strings.TrimSpace(op)
	if op == "" {
		return fmt.Errorf("operation is required")
	}
	if !LifecycleOperation(op).IsValid() {
		return fmt.Errorf("invalid operation %q (expected start, stop, or restart)", op)
	}
	return nil
}
