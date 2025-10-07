package filter

import (
	"fmt"
	"time"

	"github.com/expr-lang/expr"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/primitives"
)

const PrimitiveName = "filter"

type Spec struct {
	Expression string `json:"expression"`
}

type Filter struct{}

func (f *Filter) Name() string {
	return PrimitiveName
}

func (f *Filter) Description() string {
	return "Evaluate input data against condition. If true, input data is sent to default output branch"
}

func (f *Filter) Outputs(configuration any) []string {
	return []string{primitives.DefaultBranchName}
}

func (f *Filter) Configuration() []primitives.ConfigurationField {
	return []primitives.ConfigurationField{
		{
			Name:        "expression",
			Type:        "string",
			Description: "Boolean expression to filter data",
			Required:    true,
		},
	}
}

func (f *Filter) Execute(ctx primitives.ExecutionContext) error {
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

	outputs := map[string][]any{}
	if matches {
		outputs[primitives.DefaultBranchName] = []any{ctx.Data}
	}

	return ctx.State.Finish(outputs)
}

func (f *Filter) Actions() []primitives.Action {
	return []primitives.Action{}
}

func (f *Filter) HandleAction(ctx primitives.ActionContext) error {
	return fmt.Errorf("filter primitive does not support actions")
}
