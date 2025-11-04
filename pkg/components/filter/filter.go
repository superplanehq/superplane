package filter

import (
	"fmt"
	"time"

	"github.com/expr-lang/expr"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
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

func (f *Filter) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (f *Filter) Icon() string {
	return "funnel"
}

func (f *Filter) Color() string {
	return "red"
}

func (f *Filter) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:        "expression",
			Label:       "Filter Expression",
			Type:        components.FieldTypeString,
			Description: "Boolean expression to filter data",
			Required:    true,
		},
	}
}

func (f *Filter) Execute(ctx components.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
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
		return fmt.Errorf("expression compilation failed: %w", err)
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return fmt.Errorf("expression evaluation failed: %w", err)
	}

	matches, ok := output.(bool)
	if !ok {
		return fmt.Errorf("expression must evaluate to boolean, got %T", output)
	}

	outputs := map[string][]any{}
	if matches {
		outputs[components.DefaultOutputChannel.Name] = []any{ctx.Data}
	}

	return ctx.ExecutionStateContext.Pass(outputs)
}

func (f *Filter) Actions() []components.Action {
	return []components.Action{}
}

func (f *Filter) HandleAction(ctx components.ActionContext) error {
	return fmt.Errorf("filter does not support actions")
}

func (f *Filter) Setup(ctx components.SetupContext) error {
	return nil
}
