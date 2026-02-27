package evals

import (
	"fmt"
	"testing"

	canvasesactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
)

func TestGetDataLookupEval(t *testing.T) {
	runEvalWithInvariant(t, evalScenario{
		Prompt: `For PR cleanup, read sandbox id from stored list mapping and return sandbox_id for pull_request.`,
		Blocks: []evalBlock{
			{Name: "getData", Type: "component"},
			{Name: "daytona.deleteSandbox", Type: "component"},
		},
	}, func(plan *canvasesactions.EvalPlan) error {
		for _, op := range plan.Operations {
			if operationType(op) != "add_node" {
				continue
			}
			if addNodeBlockName(op) != "getData" {
				continue
			}

			config := operationConfig(op)
			if config == nil {
				return fmt.Errorf("getData add_node must include configuration")
			}

			mode, _ := config["mode"].(string)
			if mode == "" {
				return fmt.Errorf("getData configuration must include explicit mode")
			}

			return nil
		}

		return fmt.Errorf("expected planner to use getData for sandbox mapping retrieval")
	})
}

