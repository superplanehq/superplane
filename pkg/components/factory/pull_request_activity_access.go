package factory

import (
	"fmt"
	"math/rand"
	"strings"
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
	runs core.RunExecutionContext,
	eventType string,
	result *core.PullRequestActivityResult,
	requestedAccess string,
) error {
	if result.Outcome == core.PullRequestActivityOutcomeWaiting {
		return scheduleAcquireAccess(requests)
	}

	if err := recordExclusiveLimitReachedError(runs, requestedAccess, result); err != nil {
		return err
	}

	return state.Emit(activityOutputChannel(requestedAccess, result.Outcome), eventType, []any{activityPayload(result)})
}

func activityOutputChannel(requestedAccess, outcome string) string {
	if outcome == core.PullRequestActivityOutcomeLimitReached &&
		strings.TrimSpace(requestedAccess) == core.PullRequestActivityAccessExclusive {
		return outcome
	}
	return core.DefaultOutputChannel.Name
}

func recordExclusiveLimitReachedError(
	runs core.RunExecutionContext,
	requestedAccess string,
	result *core.PullRequestActivityResult,
) error {
	if runs == nil || result == nil {
		return nil
	}
	if result.Outcome != core.PullRequestActivityOutcomeLimitReached {
		return nil
	}
	if strings.TrimSpace(requestedAccess) != core.PullRequestActivityAccessExclusive {
		return nil
	}
	return runs.AddError(limitReachedErrorMessage(result))
}

func limitReachedErrorMessage(result *core.PullRequestActivityResult) string {
	if result.Activity != nil && result.Activity.AttemptLimit != nil && *result.Activity.AttemptLimit > 0 {
		return "Automatic fixes paused after " + attemptCountLabel(*result.Activity.AttemptLimit)
	}
	return "Automatic fixes paused after the attempt limit"
}

func attemptCountLabel(count int) string {
	if count == 1 {
		return "1 attempt"
	}
	return fmt.Sprintf("%d attempts", count)
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
