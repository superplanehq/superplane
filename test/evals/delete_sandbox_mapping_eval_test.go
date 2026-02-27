package evals

import (
	"fmt"
	"testing"

	canvasesactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
)

func TestDeleteSandboxMappingEval(t *testing.T) {
	runEvalWithInvariant(t, evalScenario{
		Prompt: `When pull request is closed, delete sandbox and then remove its mapping from canvas data.`,
		Blocks: []evalBlock{
			{Name: "github.onPullRequest", Type: "trigger"},
			{Name: "getData", Type: "component"},
			{Name: "daytona.deleteSandbox", Type: "component"},
			{Name: "clearData", Type: "component"},
		},
	}, func(plan *canvasesactions.EvalPlan) error {
		for _, op := range plan.Operations {
			if operationType(op) != "add_node" {
				continue
			}
			if addNodeBlockName(op) == "clearData" {
				return nil
			}
		}

		return fmt.Errorf("expected planner to add clearData to remove PR->sandbox mapping")
	})
}

