package evals

import (
	"fmt"
	"testing"

	canvasesactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
)

func TestPRActionSelectionEval(t *testing.T) {
	runEvalWithInvariant(t, evalScenario{
		Prompt: `Create a flow for pull request cleanup:
- trigger on pull request closed
- delete sandbox for the closed PR`,
		Blocks: []evalBlock{
			{Name: "github.onPullRequest", Type: "trigger"},
			{Name: "getData", Type: "component"},
			{Name: "daytona.deleteSandbox", Type: "component"},
			{Name: "clearData", Type: "component"},
		},
	}, func(plan *canvasesactions.EvalPlan) error {
		for _, op := range plan.Operations {
			opType := operationType(op)
			if opType != "add_node" && opType != "update_node_config" {
				continue
			}

			blockName := addNodeBlockName(op)
			actions := actionList(operationConfig(op))
			if blockName != "github.onPullRequest" && len(actions) == 0 {
				continue
			}

			if containsString(actions, "closed") {
				return nil
			}
		}

		return fmt.Errorf("expected onPullRequest actions to include 'closed'")
	})
}

