package ifp

import (
	"fmt"
	"time"

	"github.com/expr-lang/expr"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
)

const ComponentName = "if"
const BranchNameTrue = "true"
const BranchNameFalse = "false"

type If struct{}

type Spec struct {
	Expression string `json:"expression"`
}

func (f *If) Name() string {
	return ComponentName
}

func (f *If) Description() string {
	return "Evaluate input data against condition and route to true or false branches"
}

func (f *If) OutputBranches(configuration any) []string {
	return []string{BranchNameTrue, BranchNameFalse}
}

func (f *If) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:        "expression",
			Type:        "string",
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
			BranchNameTrue: {ctx.Data},
		}
	} else {
		outputs = map[string][]any{
			BranchNameFalse: {ctx.Data},
		}
	}

	return ctx.ExecutionStateContext.Finish(outputs)
}

func (f *If) Actions() []components.Action {
	return []components.Action{}
}

func (f *If) HandleAction(ctx components.ActionContext) error {
	return fmt.Errorf("if does not support actions")
}
