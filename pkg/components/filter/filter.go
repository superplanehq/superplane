package filter

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "filter"

func init() {
	registry.RegisterComponent(ComponentName, &Filter{})
}

type Spec struct {
	Expression string `json:"expression"`
}

type Filter struct{}

func (f *Filter) Name() string {
	return ComponentName
}

func (f *Filter) Label() string {
	return "Filter"
}

func (f *Filter) Description() string {
	return "Filter events based on their content"
}

func (f *Filter) Documentation() string {
	return `The Filter component evaluates a boolean expression against incoming events and only forwards events that match the condition.

## Use Cases

- **Data validation**: Only process events that meet certain criteria
- **Event filtering**: Filter out unwanted events before processing
- **Conditional routing**: Stop processing events that don't match requirements
- **Data quality**: Ensure only valid data continues through the workflow

## How It Works

1. The Filter component evaluates a boolean expression against the incoming event data
2. If the expression evaluates to ` + "`true`" + `, the event is emitted to the default output channel
3. If the expression evaluates to ` + "`false`" + `, the execution passes without emitting (effectively filtering out the event)

## Expression Environment

The expression has access to:
- **$**: The run context data
- **root()**: Access to the root event data
- **previous()**: Access to previous node outputs (optionally with depth parameter)

## Examples

- ` + "`$[\"Node Name\"].status == \"active\"`" + `: Only forward events where status is "active"
- ` + "`$[\"Node Name\"].amount > 1000`" + `: Filter events with amount greater than 1000
- ` + "`$[\"Node Name\"].user.role == \"admin\" && $[\"Node Name\"].action == \"delete\"`" + `: Complex condition checking multiple fields`
}

func (f *Filter) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (f *Filter) Icon() string {
	return "funnel"
}

func (f *Filter) Color() string {
	return "red"
}

func (f *Filter) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "expression",
			Label:       "Filter Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Boolean expression to filter data",
			Required:    true,
		},
	}
}

func (f *Filter) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	output, err := ctx.Expressions.Run(spec.Expression)
	if err != nil {
		return fmt.Errorf("expression evaluation failed: %w", err)
	}

	matches, ok := output.(bool)
	if !ok {
		return fmt.Errorf("expression must evaluate to boolean, got %T: %v", output, output)
	}

	if matches {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"filter.executed",
			[]any{map[string]any{}},
		)
	}

	return ctx.ExecutionState.Pass()
}

func (f *Filter) Actions() []core.Action {
	return []core.Action{}
}

func (f *Filter) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("filter does not support actions")
}

func (f *Filter) Setup(ctx core.SetupContext) error {
	return nil
}

func (f *Filter) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (f *Filter) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (f *Filter) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (f *Filter) Cleanup(ctx core.SetupContext) error {
	return nil
}
