package fanout

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "fanOut"
const PayloadType = "fanout.item"
const ChannelNameItem = "item"

func init() {
	registry.RegisterAction(ComponentName, &FanOut{})
}

type FanOut struct{}

type Spec struct {
	ArrayExpression string `json:"arrayExpression"`
}

func (f *FanOut) Name() string {
	return ComponentName
}

func (f *FanOut) Label() string {
	return "Fan Out"
}

func (f *FanOut) Description() string {
	return "Emit one downstream event per item in an array"
}

func (f *FanOut) Documentation() string {
	return `The Fan Out component reads an array from the upstream payload and emits one downstream event per element.

## Use Cases

- Iterate over a list of results and process each one independently
- Fan out runner output arrays into per-item workflow paths
- Process each page, service, or record with the same downstream steps

## How It Works

1. Evaluates the configured array expression against the incoming event data
2. Emits one ` + "`fanout.item`" + ` event to the ` + "`item`" + ` channel for each element
3. If the array is empty, passes without emitting any events

## Output Fields (per item)

- **item**: The array element value
- **index**: Zero-based index of the element
- **totalCount**: Total number of items in the array

## Expression Environment

- **$**: The run context data
- **root()**: Access root event data
- **previous()**: Access previous node outputs`
}

func (f *FanOut) Icon() string {
	return "split"
}

func (f *FanOut) Color() string {
	return "blue"
}

func (f *FanOut) ExampleOutput() map[string]any {
	return map[string]any{
		"item":       map[string]any{"service": "EC2", "cost_usd": 42.5},
		"index":      0,
		"totalCount": 3,
	}
}

func (f *FanOut) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameItem, Label: "Item"},
	}
}

func (f *FanOut) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "arrayExpression",
			Label:       "Array Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Expression that evaluates to the array to fan out",
			Required:    true,
		},
	}
}

func (f *FanOut) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateSpec(spec)
}

func (f *FanOut) Execute(ctx core.ExecutionContext) error {
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
	// direct []any
	if s, ok := v.([]any); ok {
		return s, nil
	}
	// reflect-based fallback for typed slices
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

func (f *FanOut) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (f *FanOut) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (f *FanOut) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (f *FanOut) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (f *FanOut) Hooks() []core.Hook {
	return []core.Hook{}
}

func (f *FanOut) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
