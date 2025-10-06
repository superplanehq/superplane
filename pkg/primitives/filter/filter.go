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

func (f *Filter) Execute(ctx primitives.ExecutionContext) (*primitives.Result, error) {
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

	if !matches {
		return nil, nil
	}

	return &primitives.Result{
		Branches: map[string][]any{
			primitives.DefaultBranchName: {ctx.Data},
		},
	}, nil
}
