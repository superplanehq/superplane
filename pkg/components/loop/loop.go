package loop

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

const ComponentName = "loop"
const PayloadType = "loop.iteration"
const ChannelNameIteration = "iteration"

const (
	ModeCollection = "collection"
	ModeCount      = "count"
	ModeRange      = "range"
)

const DefaultItemVariable = "item"

var (
	reservedItemVariables = map[string]struct{}{
		"$":          {},
		"memory":     {},
		"config":     {},
		"root":       {},
		"previous":   {},
		"ctx":        {},
		"index":      {},
		"totalCount": {},
		"first":      {},
		"last":       {},
	}

	itemVariablePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

func init() {
	registry.RegisterAction(ComponentName, &Loop{})
}

type Loop struct{}

type Spec struct {
	Mode                 string `json:"mode"`
	CollectionExpression string `json:"collectionExpression"`
	CountExpression      string `json:"countExpression"`
	StartExpression      string `json:"startExpression"`
	EndExpression        string `json:"endExpression"`
	StepExpression       string `json:"stepExpression"`
	ItemVariable         string `json:"itemVariable"`
	PayloadExpression    string `json:"payloadExpression"`
}

func (c *Loop) Name() string {
	return ComponentName
}

func (c *Loop) Label() string {
	return "Loop"
}

func (c *Loop) Description() string {
	return "Emit one downstream event per loop iteration"
}

func (c *Loop) Documentation() string {
	return `The Loop component emits one downstream event per iteration using collection, count, or range modes.

## Use Cases

- Iterate over any list in the message chain with a custom item variable
- Repeat downstream steps a fixed number of times
- Walk numeric ranges with configurable start, end, and step
- Shape per-iteration payloads with a custom expression

## Loop Modes

- **Collection**: Evaluate a list expression and emit one event per element
- **Count**: Repeat a fixed number of times (0-based index as the current value)
- **Range**: Walk from a start value to an end value using an optional step (default ` + "`1`" + `)

## How It Works

1. Resolves iterations from the selected loop mode
2. Evaluates an optional payload expression for each iteration
3. Emits one ` + "`loop.iteration`" + ` event to the ` + "`iteration`" + ` channel per iteration
4. If there are zero iterations, passes without emitting any events

## Limits

- At most ` + fmt.Sprintf("%d", core.MaxEmitCount) + ` iterations per execution. Larger loops fail with an error.

## Default Output Fields

When no payload expression is configured, each event includes:

- **itemVariable**: The current iteration value (key matches the configured item variable)
- **index**: Zero-based iteration index
- **totalCount**: Total number of iterations
- **first**: Whether this is the first iteration
- **last**: Whether this is the last iteration

## Expression Environment

Each payload expression can reference:

- **$**: The run context data
- **root()**: Access root event data
- **previous()**: Access previous node outputs
- **index**, **totalCount**, **first**, **last**: Current iteration metadata
- The configured **item variable** for the current value`
}

func (c *Loop) Icon() string {
	return "refresh-cw"
}

func (c *Loop) Color() string {
	return "indigo"
}

func (c *Loop) ExampleOutput() map[string]any {
	return exampleOutput()
}

func (c *Loop) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameIteration, Label: "Iteration"},
	}
}

func (c *Loop) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "mode",
			Label:       "Loop Mode",
			Type:        configuration.FieldTypeSelect,
			Description: "Choose how iterations are generated",
			Required:    true,
			Default:     ModeCollection,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Collection", Value: ModeCollection, Description: "Iterate over a list expression"},
						{Label: "Count", Value: ModeCount, Description: "Repeat a fixed number of times"},
						{Label: "Range", Value: ModeRange, Description: "Walk a numeric range"},
					},
				},
			},
		},
		{
			Name:        "collectionExpression",
			Label:       "Collection Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Expression that evaluates to the list to iterate over",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeCollection}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeCollection}},
			},
		},
		{
			Name:        "countExpression",
			Label:       "Count Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Expression that evaluates to a non-negative whole number",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeCount}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeCount}},
			},
		},
		{
			Name:        "startExpression",
			Label:       "Start Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Expression that evaluates to the first range value",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeRange}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeRange}},
			},
		},
		{
			Name:        "endExpression",
			Label:       "End Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Expression that evaluates to the last inclusive range value",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeRange}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeRange}},
			},
		},
		{
			Name:        "stepExpression",
			Label:       "Step Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Expression that evaluates to the range step (defaults to 1 when empty)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeRange}},
			},
		},
		{
			Name:        "itemVariable",
			Label:       "Item Variable",
			Type:        configuration.FieldTypeString,
			Description: "Name of the current iteration value in expressions and default payloads",
			Default:     DefaultItemVariable,
			TypeOptions: &configuration.TypeOptions{
				String: &configuration.StringTypeOptions{
					AllowExpressions: boolPtr(false),
				},
			},
		},
		{
			Name:        "payloadExpression",
			Label:       "Payload Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Optional expression that evaluates to the payload object for each iteration. When empty, a default payload is emitted.",
		},
	}
}

func (c *Loop) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateSpec(spec)
}

func (c *Loop) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}

	iterations, err := spec.resolveIterations(ctx.Expressions)
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(map[string]any{
		"mode":         spec.Mode,
		"count":        len(iterations),
		"itemVariable": spec.ItemVariable,
	}); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if len(iterations) == 0 {
		return ctx.ExecutionState.Pass()
	}
	if len(iterations) > core.MaxEmitCount {
		return fmt.Errorf("loop has %d iterations; Loop supports at most %d per execution", len(iterations), core.MaxEmitCount)
	}

	totalCount := len(iterations)
	payloads := make([]any, 0, totalCount)
	for _, current := range iterations {
		payload, payloadErr := spec.buildPayload(current, totalCount, ctx.Expressions)
		if payloadErr != nil {
			return payloadErr
		}
		payloads = append(payloads, payload)
	}

	return ctx.ExecutionState.Emit(ChannelNameIteration, PayloadType, payloads)
}

func (s Spec) buildPayload(current iteration, totalCount int, expressions core.ExpressionContext) (any, error) {
	variables := iterationVariables(s.ItemVariable, current, totalCount)

	if strings.TrimSpace(s.PayloadExpression) == "" {
		return defaultPayload(variables), nil
	}

	result, err := expressions.RunWithExtraVariables(s.PayloadExpression, variables)
	if err != nil {
		return nil, fmt.Errorf("payload expression evaluation failed: %w", err)
	}

	if payload, ok := result.(map[string]any); ok {
		return payload, nil
	}

	payload := defaultPayload(variables)
	payload[s.ItemVariable] = result
	return payload, nil
}

func defaultPayload(variables map[string]any) map[string]any {
	payload := make(map[string]any, len(variables))
	for key, value := range variables {
		payload[key] = value
	}
	return payload
}

func iterationVariables(itemVariable string, current iteration, totalCount int) map[string]any {
	return map[string]any{
		itemVariable: current.Value,
		"index":      current.Index,
		"totalCount": totalCount,
		"first":      current.Index == 0,
		"last":       current.Index == totalCount-1,
	}
}

func decodeSpec(raw any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(raw, &spec); err != nil {
		return Spec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	return normalizeSpec(spec), nil
}

func normalizeSpec(spec Spec) Spec {
	spec.Mode = strings.TrimSpace(spec.Mode)
	if spec.Mode == "" {
		spec.Mode = ModeCollection
	}
	spec.CollectionExpression = strings.TrimSpace(spec.CollectionExpression)
	spec.CountExpression = strings.TrimSpace(spec.CountExpression)
	spec.StartExpression = strings.TrimSpace(spec.StartExpression)
	spec.EndExpression = strings.TrimSpace(spec.EndExpression)
	spec.StepExpression = strings.TrimSpace(spec.StepExpression)
	spec.ItemVariable = strings.TrimSpace(spec.ItemVariable)
	spec.PayloadExpression = strings.TrimSpace(spec.PayloadExpression)
	if spec.ItemVariable == "" {
		spec.ItemVariable = DefaultItemVariable
	}
	return spec
}

func validateSpec(spec Spec) error {
	switch spec.Mode {
	case ModeCollection:
		if spec.CollectionExpression == "" {
			return fmt.Errorf("collectionExpression is required when mode is collection")
		}
	case ModeCount:
		if spec.CountExpression == "" {
			return fmt.Errorf("countExpression is required when mode is count")
		}
	case ModeRange:
		if spec.StartExpression == "" {
			return fmt.Errorf("startExpression is required when mode is range")
		}
		if spec.EndExpression == "" {
			return fmt.Errorf("endExpression is required when mode is range")
		}
	default:
		return fmt.Errorf("mode must be one of collection, count, or range")
	}

	if !itemVariablePattern.MatchString(spec.ItemVariable) {
		return fmt.Errorf("itemVariable %q must match %s", spec.ItemVariable, itemVariablePattern.String())
	}
	if _, ok := reservedItemVariables[spec.ItemVariable]; ok {
		return fmt.Errorf("itemVariable %q is reserved", spec.ItemVariable)
	}

	return nil
}

func boolPtr(v bool) *bool {
	return &v
}

func (c *Loop) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Loop) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *Loop) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *Loop) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *Loop) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *Loop) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
