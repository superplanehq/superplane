package foreach

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "forEach"
const PayloadType = "foreach.item"
const ChannelNameItem = "item"

func init() {
	registry.RegisterAction(ComponentName, &ForEach{})
}

type ForEach struct{}

type Spec struct {
	ArrayExpression string `json:"arrayExpression"`
}

func (c *ForEach) Name() string {
	return ComponentName
}

func (c *ForEach) Label() string {
	return "For Each"
}

func (c *ForEach) Description() string {
	return "Emit one downstream event per item in an array"
}

func (c *ForEach) Documentation() string {
	return `The For Each component reads an array from the upstream payload and emits one downstream event per element.

## Use Cases

- Iterate over a list of results and process each one independently
- Split runner output arrays into per-item workflow paths
- Process each page, service, or record with the same downstream steps

## How It Works

1. Evaluates the configured array expression against the incoming event data
2. Emits one ` + "`foreach.item`" + ` event to the ` + "`item`" + ` channel for each element
3. If the array is empty, passes without emitting any events

## Limits

- At most ` + fmt.Sprintf("%d", config.MaxEmitCount()) + ` items per execution. Larger arrays fail with an error.
- Self-hosted deployments can raise this cap with the ` + "`SUPERPLANE_MAX_EMIT_COUNT`" + ` environment variable.

## Output Fields (per item)

- **item**: The array element value
- **index**: Zero-based index of the element
- **totalCount**: Total number of items in the array

## Expression Environment

- **$**: The run context data
- **root()**: Access root event data
- **previous()**: Access previous node outputs`
}

func (c *ForEach) Icon() string {
	return "repeat"
}

func (c *ForEach) Color() string {
	return "blue"
}

func (c *ForEach) ExampleOutput() map[string]any {
	return exampleOutput()
}

func (c *ForEach) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameItem, Label: "Item"},
	}
}

func (c *ForEach) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "arrayExpression",
			Label:       "Array Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Expression that evaluates to the array to iterate over",
			Required:    true,
		},
	}
}

func (c *ForEach) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateSpec(spec)
}

func (c *ForEach) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}

	result, err := ctx.Expressions.Run(spec.ArrayExpression)
	if err != nil {
		return fmt.Errorf("expression evaluation failed: %w", err)
	}

	items, err := toSlice(result)
	if err != nil {
		return fmt.Errorf("expression must evaluate to an array: %w", err)
	}

	if err := ctx.Metadata.Set(map[string]any{
		"arrayExpression": spec.ArrayExpression,
		"count":           len(items),
	}); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if len(items) == 0 {
		return ctx.ExecutionState.Pass()
	}
	maxEmitCount := config.MaxEmitCount()
	if len(items) > maxEmitCount {
		return fmt.Errorf("array has %d items; For Each supports at most %d items per execution", len(items), maxEmitCount)
	}

	payloads := make([]any, 0, len(items))
	for i, item := range items {
		payloads = append(payloads, map[string]any{
			"item":       item,
			"index":      i,
			"totalCount": len(items),
		})
	}

	return ctx.ExecutionState.Emit(ChannelNameItem, PayloadType, payloads)
}

func decodeSpec(raw any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(raw, &spec); err != nil {
		return Spec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	return spec, nil
}

func validateSpec(spec Spec) error {
	if spec.ArrayExpression == "" {
		return fmt.Errorf("arrayExpression is required")
	}
	return nil
}

func toSlice(v any) ([]any, error) {
	if v == nil {
		return []any{}, nil
	}
	if s, ok := v.([]any); ok {
		return s, nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("got %T", v)
	}
	result := make([]any, rv.Len())
	for i := range result {
		result[i] = rv.Index(i).Interface()
	}
	return result, nil
}

func (c *ForEach) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ForEach) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ForEach) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ForEach) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ForEach) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ForEach) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
