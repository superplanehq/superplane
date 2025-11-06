package ifp

import (
	"fmt"
	"time"

	"github.com/expr-lang/expr"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
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

func (f *If) Icon() string {
	return "split"
}

func (f *If) Color() string {
	return "red"
}

func (f *If) IsUserVisible() bool { return true }

func (f *If) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{
		{Name: "true", Label: "True"},
		{Name: "false", Label: "False"},
	}
}

func (f *If) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "expression",
			Type:        configuration.FieldTypeString,
			Description: "Boolean expression to evaluate",
			Required:    true,
		},
	}
}

func (f *If) Execute(ctx components.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	env := map[string]any{
		"$": ctx.Data,
	}

	vm, err := expr.Compile(spec.Expression, []expr.Option{
		expr.Env(env),
		expr.AsBool(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
	}...)

	if err != nil {
		return err
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return fmt.Errorf("expression evaluation failed: %w", err)
	}

	matches, ok := output.(bool)
	if !ok {
		return fmt.Errorf("expression must evaluate to boolean, got %T", output)
	}

	var outputs map[string][]any
	if matches {
		outputs = map[string][]any{
			ChannelNameTrue: {ctx.Data},
		}
	} else {
		outputs = map[string][]any{
			ChannelNameFalse: {ctx.Data},
		}
	}

	return ctx.ExecutionStateContext.Pass(outputs)
}

func (f *If) Actions() []components.Action {
	return []components.Action{}
}

func (f *If) HandleAction(ctx components.ActionContext) error {
	return fmt.Errorf("if does not support actions")
}

func (f *If) Setup(ctx components.SetupContext) error {
	return nil
}
