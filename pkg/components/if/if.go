package ifp

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "if"
const ChannelNameTrue = "true"
const ChannelNameFalse = "false"

func init() {
	registry.RegisterComponent(ComponentName, &If{})
}

type If struct{}

type Spec struct {
	Expression string `json:"expression"`
}

func (f *If) Name() string {
	return ComponentName
}

func (f *If) Label() string {
	return "If"
}

func (f *If) Description() string {
	return "Route events based on expression"
}

func (f *If) Documentation() string {
	return `The If component evaluates a boolean expression and routes events to different output channels based on the result.

## Use Cases

- **Conditional branching**: Route events down different paths based on conditions
- **Decision logic**: Implement if-then-else logic in workflows
- **Data routing**: Send events to different processing paths
- **Workflow control**: Control workflow flow based on event properties

## How It Works

1. The If component evaluates a boolean expression against the incoming event data
2. If the expression evaluates to ` + "`true`" + `, the event is emitted to the "True" output channel
3. If the expression evaluates to ` + "`false`" + `, the event is emitted to the "False" output channel

## Output Channels

- **True**: Events where the expression evaluates to ` + "`true`" + `
- **False**: Events where the expression evaluates to ` + "`false`" + `

## Expression Environment

The expression has access to:
- **$**: The run context data
- **root()**: Access to the root event data
- **previous()**: Access to previous node outputs (optionally with depth parameter)

## Examples

- ` + "`$[\"Node Name\"].status == \"approved\"`" + `: Route approved items to True channel
- ` + "`$[\"Node Name\"].amount > 1000`" + `: Route high-value items to True channel
- ` + "`$[\"Node Name\"].user.role == \"admin\"`" + `: Route admin actions to True channel`
}

func (f *If) Icon() string {
	return "split"
}

func (f *If) Color() string {
	return "red"
}

func (f *If) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "true", Label: "True"},
		{Name: "false", Label: "False"},
	}
}

func (f *If) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Boolean expression to evaluate",
			Required:    true,
		},
	}
}

func (f *If) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	output, err := ctx.Expressions.Run(spec.Expression)
	if err != nil {
		return err
	}

	matches, ok := output.(bool)
	if !ok {
		return fmt.Errorf("expression must evaluate to boolean, got %T: %v", output, output)
	}

	if matches {
		return ctx.ExecutionState.Emit(
			ChannelNameTrue,
			"if.executed",
			[]any{map[string]any{}},
		)
	}

	return ctx.ExecutionState.Emit(
		ChannelNameFalse,
		"if.executed",
		[]any{map[string]any{}},
	)
}

func (f *If) Actions() []core.Action {
	return []core.Action{}
}

func (f *If) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("if does not support actions")
}

func (f *If) Setup(ctx core.SetupContext) error {
	return nil
}

func (f *If) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (f *If) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (f *If) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (f *If) Cleanup(ctx core.SetupContext) error {
	return nil
}
