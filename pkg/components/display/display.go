package display

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "display"
const PayloadType = "display.executed"
const DefaultColor = "gray"

var expressionRegex = regexp.MustCompile(`\{\{(.*?)\}\}`)

func init() {
	registry.RegisterAction(ComponentName, &Display{})
}

type Display struct{}

type Spec struct {
	Value string `json:"value"`
	Color string `json:"color"`
}

type Result struct {
	Value string `json:"value"`
	Color string `json:"color"`
}

func (c *Display) Name() string {
	return ComponentName
}

func (c *Display) Label() string {
	return "Display"
}

func (c *Display) Description() string {
	return "Render a value and color badge from the latest execution"
}

func (c *Display) Documentation() string {
	return `The Display component resolves a value and a color from the current run payload and stores the result in execution metadata for canvas rendering.

## Behavior

1. Resolves **Value** against the run payload (supports ` + "`{{ ... }}`" + ` expressions)
2. Resolves **Color** against the run payload (supports ` + "`{{ ... }}`" + ` expressions)
3. Stores ` + "`display_result`" + ` in execution metadata as {value, color}
4. Emits the incoming payload to the default output channel unchanged

## Error Handling

Expression errors never fail the run. If resolving either field fails, the component stores:

- ` + "`value`" + `: ` + "`[expression error: <message>]`" + `
- ` + "`color`" + `: ` + "`gray`" + `

## Supported Colors

- ` + "`green`" + `
- ` + "`yellow`" + `
- ` + "`red`" + `
- ` + "`blue`" + `
- ` + "`gray`" + ` (default)`
}

func (c *Display) Icon() string {
	return "tag"
}

func (c *Display) Color() string {
	return "gray"
}

func (c *Display) ExampleOutput() map[string]any {
	return ExampleOutput()
}

func (c *Display) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *Display) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeText,
			Description: "Text to display. Supports {{ }} expressions.",
			Required:    true,
		},
		{
			Name:        "color",
			Label:       "Color",
			Type:        configuration.FieldTypeText,
			Description: "Color for the badge (green, yellow, red, blue, gray). Supports {{ }} expressions.",
			Required:    false,
			Default:     DefaultColor,
		},
	}
}

func (c *Display) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	displayResult := Result{
		Value: spec.Value,
		Color: DefaultColor,
	}

	value, err := resolveField(ctx.Expressions, spec.Value)
	if err != nil {
		displayResult = expressionErrorResult(err)
	} else {
		displayResult.Value = value
		colorInput := strings.TrimSpace(spec.Color)
		if colorInput == "" {
			colorInput = DefaultColor
		}

		color, colorErr := resolveField(ctx.Expressions, colorInput)
		if colorErr != nil {
			displayResult = expressionErrorResult(colorErr)
		} else {
			displayResult.Color = normalizeColor(color)
		}
	}

	if err := ctx.Metadata.Set(mergeDisplayResult(ctx.Metadata.Get(), displayResult)); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{ctx.Data},
	)
}

func resolveField(expressions core.ExpressionContext, input string) (string, error) {
	if !expressionRegex.MatchString(input) {
		return input, nil
	}

	if expressions == nil {
		return "", fmt.Errorf("expression context is not available")
	}

	var expressionErr error
	result := expressionRegex.ReplaceAllStringFunc(input, func(match string) string {
		matches := expressionRegex.FindStringSubmatch(match)
		if len(matches) != 2 {
			return match
		}

		expression := strings.TrimSpace(matches[1])
		value, err := expressions.Run(expression)
		if err != nil {
			expressionErr = err
			return ""
		}

		return fmt.Sprintf("%v", value)
	})

	if expressionErr != nil {
		return "", expressionErr
	}

	return result, nil
}

func normalizeColor(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "green", "yellow", "red", "blue", "gray":
		return normalized
	default:
		return DefaultColor
	}
}

func expressionErrorResult(err error) Result {
	return Result{
		Value: fmt.Sprintf("[expression error: %s]", err.Error()),
		Color: DefaultColor,
	}
}

func mergeDisplayResult(existing any, result Result) map[string]any {
	metadata, ok := existing.(map[string]any)
	if !ok || metadata == nil {
		metadata = map[string]any{}
	} else {
		merged := make(map[string]any, len(metadata)+1)
		for k, v := range metadata {
			merged[k] = v
		}
		metadata = merged
	}

	metadata["display_result"] = map[string]any{
		"value": result.Value,
		"color": result.Color,
	}
	return metadata
}

func (c *Display) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Display) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *Display) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *Display) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *Display) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *Display) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *Display) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
