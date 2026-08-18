package common

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	ReactionRetryHookName = "retryReaction"

	reactionMaxAttempts = 3
	reactionBaseDelay   = 5 * time.Second
	reactionMaxDelay    = 5 * time.Minute
)

var supportedReactionContents = map[string]struct{}{
	"+1":       {},
	"-1":       {},
	"laugh":    {},
	"confused": {},
	"heart":    {},
	"hooray":   {},
	"rocket":   {},
	"eyes":     {},
}

type ReactionOperation func() (*github.Reaction, *github.Response, error)

type reactionRetryMetadata struct {
	Attempts    int    `json:"attempts" mapstructure:"attempts"`
	LastError   string `json:"lastError" mapstructure:"lastError"`
	LastStatus  *int   `json:"lastStatus,omitempty" mapstructure:"lastStatus"`
	NextRetryAt string `json:"nextRetryAt" mapstructure:"nextRetryAt"`
}

func ValidateReactionContent(content string) error {
	if _, ok := supportedReactionContents[content]; !ok {
		return fmt.Errorf("invalid reaction content: %s", content)
	}

	return nil
}

// ExecuteReaction executes an idempotent GitHub reaction request. Transient
// provider failures are scheduled through the durable action-hook mechanism so
// the database transaction is not held open while waiting between attempts.
func ExecuteReaction(ctx core.ExecutionContext, operation ReactionOperation) error {
	return executeReaction(
		ctx.Metadata,
		ctx.Requests,
		ctx.ExecutionState,
		operation,
		1,
		false,
	)
}

func RetryReaction(ctx core.ActionHookContext, operation ReactionOperation) error {
	if ctx.Name != ReactionRetryHookName {
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata reactionRetryMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return FailReactionHook(ctx, fmt.Errorf("failed to decode reaction retry metadata: %w", err))
	}

	if metadata.Attempts < 1 || metadata.Attempts >= reactionMaxAttempts {
		return FailReactionHook(ctx, fmt.Errorf("invalid reaction retry attempt: %d", metadata.Attempts))
	}

	return executeReaction(
		ctx.Metadata,
		ctx.Requests,
		ctx.ExecutionState,
		operation,
		metadata.Attempts+1,
		true,
	)
}

func FailReactionHook(ctx core.ActionHookContext, err error) error {
	return ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, err.Error())
}

func ReactionHooks() []core.Hook {
	return []core.Hook{{Name: ReactionRetryHookName, Type: core.HookTypeInternal}}
}

func executeReaction(
	metadataCtx core.MetadataWriter,
	requestCtx core.RequestContext,
	executionStateCtx core.ExecutionStateContext,
	operation ReactionOperation,
	attempt int,
	finishOnTerminalFailure bool,
) error {
	reaction, response, err := operation()
	if err == nil {
		return executionStateCtx.Emit(
			core.DefaultOutputChannel.Name,
			"github.reaction",
			[]any{reaction},
		)
	}

	if !isTransientReactionError(response) {
		if finishOnTerminalFailure {
			return executionStateCtx.Fail(models.CanvasNodeExecutionResultReasonError, err.Error())
		}
		return err
	}

	if attempt >= reactionMaxAttempts {
		terminalErr := fmt.Errorf("reaction request failed after %d attempts: %w", attempt, err)
		if finishOnTerminalFailure {
			return executionStateCtx.Fail(models.CanvasNodeExecutionResultReasonError, terminalErr.Error())
		}
		return terminalErr
	}

	delay := reactionRetryDelay(response, attempt, time.Now())
	nextRetryAt := time.Now().Add(delay).Format(time.RFC3339)
	metadata := reactionRetryMetadata{
		Attempts:    attempt,
		LastError:   err.Error(),
		NextRetryAt: nextRetryAt,
	}
	if response != nil && response.Response != nil {
		status := response.StatusCode
		metadata.LastStatus = &status
	}

	if err := metadataCtx.Set(metadata); err != nil {
		return fmt.Errorf("failed to set reaction retry metadata: %w", err)
	}

	if err := requestCtx.ScheduleActionCall(ReactionRetryHookName, map[string]any{}, delay); err != nil {
		return fmt.Errorf("failed to schedule reaction retry: %w", err)
	}

	return nil
}

func isTransientReactionError(response *github.Response) bool {
	if response == nil || response.Response == nil {
		return true
	}

	status := response.StatusCode
	if status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError {
		return true
	}

	if status != http.StatusForbidden {
		return false
	}

	headers := response.Header
	return headers.Get("Retry-After") != "" || headers.Get("X-RateLimit-Remaining") == "0"
}

func reactionRetryDelay(response *github.Response, attempt int, now time.Time) time.Duration {
	if response != nil && response.Response != nil {
		headers := response.Header
		if delay, ok := parseRetryAfter(headers.Get("Retry-After"), now); ok {
			return minimumReactionRetryDelay(delay)
		}

		if reset, err := strconv.ParseInt(headers.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			return minimumReactionRetryDelay(time.Unix(reset, 0).Sub(now))
		}
	}

	delay := reactionBaseDelay * time.Duration(1<<(attempt-1))
	return clampReactionRetryDelay(delay)
}

func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	if value == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second, true
	}

	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}

	return retryAt.Sub(now), true
}

func clampReactionRetryDelay(delay time.Duration) time.Duration {
	delay = minimumReactionRetryDelay(delay)

	if delay > reactionMaxDelay {
		return reactionMaxDelay
	}

	return delay
}

func minimumReactionRetryDelay(delay time.Duration) time.Duration {
	if delay < time.Second {
		return time.Second
	}

	return delay
}
