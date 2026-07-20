package runs

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const addRunErrorComponentName = "addRunError"
const addRunErrorPayloadType = "addRunError.finished"

func init() {
	registry.RegisterAction(addRunErrorComponentName, &AddRunError{})
}

type AddRunError struct{}

type addRunErrorSpec struct {
	Message string `json:"message" mapstructure:"message"`
}

func (c *AddRunError) Name() string {
	return addRunErrorComponentName
}

func (c *AddRunError) Label() string {
	return "Add Run Error"
}

func (c *AddRunError) Description() string {
	return "Record a business failure on the current run"
}

func (c *AddRunError) Documentation() string {
	return `The Add Run Error component appends an error entry to the current run.

## Use Cases

- **Channel-based failures**: Mark the run as failed when a component emits on a failed output channel instead of failing the execution
- **Handled failures**: Record that a business operation failed even though downstream nodes continue
- **Parallel branches**: Each branch can report its own failure; all entries are collected on the run

## Behavior

- Appends an error message to the run
- A run with one or more recorded errors is treated as failed when it finishes
- Downstream nodes still receive an event on the default output channel
- The execution itself passes unless storing the error exceeds platform limits`
}

func (c *AddRunError) Icon() string {
	return "circle-alert"
}

func (c *AddRunError) Color() string {
	return "gray"
}

func (c *AddRunError) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-07-20T12:00:00Z",
		"type":      addRunErrorPayloadType,
		"data": map[string]any{
			"message": "pipeline failed",
		},
	}
}

func (c *AddRunError) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddRunError) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "message",
			Label:       "Message",
			Description: "Error message to record on the run. Supports {{ }} expressions.",
			Type:        configuration.FieldTypeText,
			Required:    true,
		},
	}
}

func (c *AddRunError) Execute(ctx core.ExecutionContext) error {
	spec := addRunErrorSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("add run error: decode configuration: %w", err)
	}

	if spec.Message == "" {
		return fmt.Errorf("add run error: message is required")
	}

	if err := ctx.Runs.AddError(spec.Message); err != nil {
		if errors.Is(err, models.ErrRunErrorsTooLarge) || errors.Is(err, models.ErrRunErrorsTooMany) {
			return ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, err.Error())
		}

		return fmt.Errorf("add run error: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		addRunErrorPayloadType,
		[]any{map[string]any{
			"message": spec.Message,
		}},
	)
}

func (c *AddRunError) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddRunError) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *AddRunError) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddRunError) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddRunError) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddRunError) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddRunError) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
