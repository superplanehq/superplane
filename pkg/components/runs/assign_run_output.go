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

const assignRunOutputComponentName = "assignRunOutput"
const assignRunOutputPayloadType = "assignRunOutput.finished"

func init() {
	registry.RegisterAction(assignRunOutputComponentName, &AssignRunOutput{})
}

type AssignRunOutput struct{}

type assignRunOutputSpec struct {
	Output map[string]any `json:"output" mapstructure:"output"`
}

func (c *AssignRunOutput) Name() string {
	return assignRunOutputComponentName
}

func (c *AssignRunOutput) Label() string {
	return "Assign Run Output"
}

func (c *AssignRunOutput) Description() string {
	return "Shallow-merge values into the run output returned to callers"
}

func (c *AssignRunOutput) Documentation() string {
	return `The Assign Run Output component shallow-merges a JSON object into the current run's accumulated output.

## Use Cases

- **Callable apps**: Expose structured results to parent apps using Run App
- **Parallel branches**: Each branch can contribute different top-level keys
- **Pipeline results**: Attach deployment metadata, URLs, or identifiers for downstream consumers

## Behavior

Each execution merges its configured output object into the run using Object.assign semantics:

- Top-level keys from this component replace existing keys with the same name
- Nested objects are replaced as a whole; they are not merged recursively
- The merged run output must stay within the platform payload size limit

Downstream nodes still receive an event on the default output channel. The run result is not changed.`
}

func (c *AssignRunOutput) Icon() string {
	return "file-output"
}

func (c *AssignRunOutput) Color() string {
	return "gray"
}

func (c *AssignRunOutput) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-07-20T12:00:00Z",
		"type":      assignRunOutputPayloadType,
		"data": map[string]any{
			"output": map[string]any{
				"deploy": map[string]any{
					"id": "d-1",
				},
			},
		},
	}
}

func (c *AssignRunOutput) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AssignRunOutput) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "output",
			Label:       "Output",
			Description: "JSON object to shallow-merge into the run output. Supports {{ }} expressions.",
			Type:        configuration.FieldTypeObject,
			Required:    true,
		},
	}
}

func (c *AssignRunOutput) Execute(ctx core.ExecutionContext) error {
	spec := assignRunOutputSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("assign run output: decode configuration: %w", err)
	}

	if spec.Output == nil {
		return fmt.Errorf("assign run output: output is required")
	}

	if err := ctx.Runs.AssignOutput(spec.Output); err != nil {
		if errors.Is(err, models.ErrRunOutputTooLarge) {
			return ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, err.Error())
		}

		return fmt.Errorf("assign run output: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		assignRunOutputPayloadType,
		[]any{map[string]any{"output": spec.Output}},
	)
}

func (c *AssignRunOutput) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AssignRunOutput) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *AssignRunOutput) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AssignRunOutput) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AssignRunOutput) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AssignRunOutput) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AssignRunOutput) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
