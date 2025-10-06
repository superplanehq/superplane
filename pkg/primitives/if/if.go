package ifp

import (
	"fmt"
	"time"

	"github.com/expr-lang/expr"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/primitives"
)

const PrimitiveName = "if"
const BranchNameTrue = "true"
const BranchNameFalse = "false"

type If struct{}

type Spec struct {
	Expression string `json:"expression"`
}

func (f *If) Name() string {
	return PrimitiveName
}

func (f *If) Description() string {
	return "Evaluate input data against condition and route to true or false branches"
}

func (f *If) Outputs(configuration any) []string {
	return []string{BranchNameTrue, BranchNameFalse}
}

func (f *If) Configuration() []primitives.ConfigurationField {
	return []primitives.ConfigurationField{
		{
			Name:        "expression",
			Type:        "string",
			Description: "Boolean expression to evaluate",
			Required:    true,
		},
	}
}

func (f *If) Execute(ctx primitives.ExecutionContext) (*primitives.Result, error) {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return nil, fmt.Errorf("expression evaluation failed: %w", err)
	}

	matches, ok := output.(bool)
	if !ok {
		return nil, fmt.Errorf("expression must evaluate to boolean, got %T", output)
	}

	if matches {
		return &primitives.Result{
			Branches: map[string][]any{
				BranchNameTrue: {ctx.Data},
			},
		}, nil
	}

	return &primitives.Result{
		Branches: map[string][]any{
			BranchNameFalse: {ctx.Data},
		},
	}, nil
}
