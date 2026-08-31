package factory

import (
	"math/rand"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const acquireAccessHookName = "acquireAccess"

var acquireAccessHook = core.Hook{
	Type: core.HookTypeInternal,
	Name: acquireAccessHookName,
}

func pullRequestActivityChannels() []core.OutputChannel {
	return []core.OutputChannel{
		core.DefaultOutputChannel,
		{Name: core.PullRequestActivityOutcomeLimitReached, Label: "Limit reached"},
	}
}

func finishPullRequestActivity(
	state core.ExecutionStateContext,
	requests core.RequestContext,
	eventType string,
	result *core.PullRequestActivityResult,
) error {
	if result.Outcome == core.PullRequestActivityOutcomeWaiting {
		return scheduleAcquireAccess(requests)
	}

	channel := core.DefaultOutputChannel.Name
	if result.Outcome != core.PullRequestActivityOutcomeReady {
		channel = result.Outcome
	}

	return state.Emit(channel, eventType, []any{activityPayload(result)})
}

func scheduleAcquireAccess(requests core.RequestContext) error {
	jitter := time.Duration(rand.Intn(2000)) * time.Millisecond
	return requests.ScheduleActionCall(acquireAccessHookName, map[string]any{}, 10*time.Second+jitter)
}

func activityPayload(result *core.PullRequestActivityResult) map[string]any {
	payload := map[string]any{
		"pullRequest": result.PullRequest,
		"workOrder":   result.WorkOrder,
	}
	if result.Activity != nil {
		payload["activity"] = result.Activity
		payload["description"] = result.Activity.Description
		if result.Activity.Attempt != nil {
			payload["attempt"] = *result.Activity.Attempt
		}
		if result.Activity.AttemptLimit != nil {
			payload["attemptLimit"] = *result.Activity.AttemptLimit
		}
	}
	if result.CurrentRevision != nil {
		payload["currentRevision"] = result.CurrentRevision
	}
	if result.CurrentHeadSHA != "" {
		payload["currentHead"] = result.CurrentHeadSHA
	}
	return payload
}
