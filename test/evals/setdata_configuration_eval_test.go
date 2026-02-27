package evals

import (
	"fmt"
	"testing"

	canvasesactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
)

func TestSetDataConfigurationEval(t *testing.T) {
	runEvalWithInvariant(t, evalScenario{
		Prompt: `When a sandbox is created for a PR, store mapping in canvas data.
Use setData with append/upsert semantics for PR key and sandbox id.`,
		Blocks: []evalBlock{
			{Name: "setData", Type: "component"},
			{Name: "daytona.createSandbox", Type: "component"},
		},
	}, func(plan *canvasesactions.EvalPlan) error {
		for _, op := range plan.Operations {
			if operationType(op) != "add_node" {
				continue
			}
			if addNodeBlockName(op) != "setData" {
				continue
			}

			config := operationConfig(op)
			if config == nil {
				return fmt.Errorf("setData add_node must include configuration")
			}

			valueListRaw, ok := config["valueList"]
			if !ok {
				return fmt.Errorf("setData configuration must include valueList")
			}

			valueList, ok := valueListRaw.([]any)
			if !ok || len(valueList) == 0 {
				return fmt.Errorf("setData valueList must be a non-empty list")
			}

			return nil
		}

		return fmt.Errorf("expected planner to use setData for sandbox mapping persistence")
	})
}

