package evals

import (
	"fmt"
	"testing"

	canvasesactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
)

func TestGetDataEmitEachItemEval(t *testing.T) {
	runEvalWithInvariant(t, evalScenario{
		Prompt: `Delete every sandbox that we have stored in canvas data.`,
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

			emitEachItem, ok := config["emitEachItem"].(bool)
			if !ok || !emitEachItem {
				return fmt.Errorf("getData must set emitEachItem=true for delete-all fan-out")
			}

			itemField, _ := config["itemField"].(string)
			if itemField == "" {
				return fmt.Errorf("getData must set itemField for delete-all fan-out")
			}

			return nil
		}

		return fmt.Errorf("expected planner to add getData for delete-all sandbox retrieval")
	})
}

