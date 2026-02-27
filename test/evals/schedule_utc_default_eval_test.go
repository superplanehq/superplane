package evals

import (
	"fmt"
	"testing"

	canvasesactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
)

func TestScheduleUTCDefaultEval(t *testing.T) {
	runEvalWithInvariant(t, evalScenario{
		Prompt: `Finally, every night at 9:00 delete every sandbox that we have.`,
		Blocks: []evalBlock{
			{Name: "schedule", Type: "trigger"},
			{Name: "getData", Type: "component"},
			{Name: "daytona.deleteSandbox", Type: "component"},
		},
	}, func(plan *canvasesactions.EvalPlan) error {
		for _, op := range plan.Operations {
			if operationType(op) != "add_node" {
				continue
			}
			if addNodeBlockName(op) != "schedule" {
				continue
			}

			config := operationConfig(op)
			if config == nil {
				return fmt.Errorf("schedule add_node must include configuration")
			}

			timezone, _ := config["timezone"].(string)
			if timezone != "0" {
				return fmt.Errorf("schedule timezone must default to UTC '0', got %q", timezone)
			}

			return nil
		}

		return fmt.Errorf("expected planner to add a schedule trigger")
	})
}

