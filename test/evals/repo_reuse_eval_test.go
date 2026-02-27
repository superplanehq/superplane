package evals

import (
	"fmt"
	"testing"

	canvasesactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
)

func TestRepoReuseEval(t *testing.T) {
	runEvalWithInvariant(t, evalScenario{
		Prompt: `Previous assistant question: "Which repository should I use?"
User reply: "front"
Now continue and build the workflow without asking for repository again.`,
		Blocks: []evalBlock{
			{Name: "github.onPRComment", Type: "trigger"},
			{Name: "daytona.createSandbox", Type: "component"},
		},
	}, func(plan *canvasesactions.EvalPlan) error {
		if hasAskFollowupQuestion(plan.Operations) {
			return fmt.Errorf("should not ask follow-up question for repository short reply")
		}
		return nil
	})
}

