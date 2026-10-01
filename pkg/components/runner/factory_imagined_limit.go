package runner

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// FactoryImaginedLimitPrompt tells a factory coding agent not to invent a
// budget or a remaining context size. The application does not set one
// unless the task text sets one.
const FactoryImaginedLimitPrompt = "Do not invent a token budget, a time budget, a step limit, or a remaining context size. The application does not set one unless the task text sets one. Do not stop early, skip work, or change the setup because of an imagined limit. If the task text sets a limit, follow that limit."

// AppendFactoryImaginedLimitPrompt returns a copy of steps with
// FactoryImaginedLimitPrompt added to each prompt step. A canvas that is
// not a factory app, and a planning session, are left unchanged. The
// original steps remain suitable for live-log previews and saved prompts.
func AppendFactoryImaginedLimitPrompt(ctx core.ExecutionContext, steps []AgentStep) []AgentStep {
	if !shouldAppendFactoryImaginedLimitPrompt(ctx) {
		return steps
	}
	return appendImaginedLimitPrompt(steps)
}

func shouldAppendFactoryImaginedLimitPrompt(ctx core.ExecutionContext) bool {
	if !canvasBelongsToFactory(ctx) {
		return false
	}
	return !runBelongsToPlanningSession(ctx)
}

func runBelongsToPlanningSession(ctx core.ExecutionContext) bool {
	if ctx.RunID == uuid.Nil {
		return false
	}
	_, err := models.FindPlanningSessionByRun(database.DB(context.Background()), ctx.RunID)
	return err == nil
}

func appendImaginedLimitPrompt(steps []AgentStep) []AgentStep {
	result := append([]AgentStep(nil), steps...)
	for i := range result {
		if NormalizeAgentStepType(result[i].Type) != AgentStepPrompt || result[i].Prompt == nil {
			continue
		}
		prompt := strings.TrimSpace(*result[i].Prompt)
		if strings.Contains(prompt, FactoryImaginedLimitPrompt) {
			continue
		}
		combined := FactoryImaginedLimitPrompt
		if prompt != "" {
			combined = prompt + "\n\n" + FactoryImaginedLimitPrompt
		}
		result[i].Prompt = &combined
	}
	return result
}
