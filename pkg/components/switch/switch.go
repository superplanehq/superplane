package switchp

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
)

type Branch struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

type Spec struct {
	Branches []Branch `json:"branches"`
}

type Switch struct{}

func (s *Switch) Name() string {
	return "switch"
}

func (s *Switch) Description() string {
	return "Evaluate input data against conditions and route to the first branch where expression evaluates to true"
}

func (s *Switch) Outputs(configuration any) []string {
	spec := Spec{}
	err := mapstructure.Decode(configuration, &spec)
	if err != nil || len(spec.Branches) == 0 {
		return []string{components.DefaultBranchName}
	}

	outputs := make([]string, len(spec.Branches))
	for i, branch := range spec.Branches {
		outputs[i] = branch.Name
	}
	return outputs
}

func (s *Switch) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:        "branches",
			Type:        "array",
			Description: "Array of branch objects with name and expression fields",
			Required:    true,
		},
	}
}

func (s *Switch) Execute(ctx components.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	env := map[string]any{
		"$": ctx.Data,
	}

	outputs := make(map[string][]any)

	for _, branch := range spec.Branches {
		vm, err := expr.Compile(branch.Expression, expr.Env(env), expr.AsBool())
		if err != nil {
			return err
		}

		output, err := expr.Run(vm, env)
		if err != nil {
			return fmt.Errorf("branch %s evaluation failed: %w", branch.Name, err)
		}

		matches, ok := output.(bool)
		if !ok {
			return fmt.Errorf("branch %s expression must evaluate to boolean, got %T", branch.Name, output)
		}

		if matches {
			outputs[branch.Name] = []any{ctx.Data}
		}
	}

	return ctx.State.Finish(outputs)
}

func (s *Switch) Actions() []components.Action {
	return []components.Action{}
}

func (s *Switch) HandleAction(ctx components.ActionContext) error {
	return fmt.Errorf("switch does not support actions")
}
