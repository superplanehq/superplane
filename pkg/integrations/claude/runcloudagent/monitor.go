package runcloudagent

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runagent"
)

// HandleWebhook — Managed Agents completion is observed via polling, not webhooks.
func (a *RunCloudAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (a *RunCloudAgent) Hooks() []core.Hook {
	return []core.Hook{{
		Name: "poll",
		Type: core.HookTypeInternal,
	}}
}

func (a *RunCloudAgent) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == "poll" {
		return a.poll(ctx)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (a *RunCloudAgent) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := ExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}
	if metadata.Session == nil || metadata.Session.ID == "" {
		return nil
	}
	sessionID := metadata.Session.ID
	attempt, errs := parsePollParams(ctx)

	client, err := runagent.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		// A client/config failure is an error, not a timeout.
		return a.handleClientError(ctx, sessionID, attempt, errs, err)
	}

	sess, err := client.GetManagedSession(sessionID)
	if err != nil {
		return a.handlePollError(ctx, client, sessionID, attempt, errs)
	}

	// Always check for a terminal session before declaring a timeout: the
	// session may have finished after the previous poll saw it running.
	if sess != nil && isSessionTerminal(sess.Status) {
		return a.handleTerminalSession(ctx, client, &metadata, sess, attempt, errs)
	}

	// Still running (or status unknown).
	if attempt > maxPollAttempts {
		return a.finishTimeout(ctx, client, sessionID)
	}
	return a.scheduleNextPoll(ctx, attempt+1, 0)
}

// parsePollParams reads the attempt and error counters from the hook parameters.
func parsePollParams(ctx core.ActionHookContext) (int, int) {
	attempt, errs := 1, 0
	if a, ok := ctx.Parameters["attempt"].(float64); ok {
		attempt = int(a)
	}
	if e, ok := ctx.Parameters["errors"].(float64); ok {
		errs = int(e)
	}
	return attempt, errs
}

// finishTimeout reclaims the still-running session and emits a timeout payload.
// Cleanup runs before the emit so the session is reclaimed even if the emit fails.
func (a *RunCloudAgent) finishTimeout(ctx core.ActionHookContext, client *runagent.Client, sessionID string) error {
	ctx.Logger.Errorf("Managed session %s exceeded max poll attempts", sessionID)
	cleanupManagedSessionFromHook(client, ctx, sessionID, true)
	out := buildOutput("timeout", sessionID)
	return ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out})
}

// handlePollError records a failed status read and, once the retry budget is
// exhausted, reclaims the session and emits an error payload. Cleanup runs
// before the emit so the session is reclaimed even if the emit fails.
func (a *RunCloudAgent) handlePollError(ctx core.ActionHookContext, client *runagent.Client, sessionID string, attempt, errs int) error {
	errs++
	if errs < maxPollErrors {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}

	ctx.Logger.Errorf("Managed session %s: polling failed repeatedly", sessionID)
	cleanupManagedSessionFromHook(client, ctx, sessionID, true)
	out := buildOutput("error", sessionID)
	return ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out})
}

// handleClientError handles a failure to build the API client during polling.
// It retries a few times, then emits an error (not a timeout) so integration or
// configuration problems are surfaced accurately. Without a client the session
// cannot be reclaimed here; a persisted session id is cleaned up on Cancel.
func (a *RunCloudAgent) handleClientError(ctx core.ActionHookContext, sessionID string, attempt, errs int, cause error) error {
	errs++
	if errs < maxPollErrors {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}
	ctx.Logger.Errorf("Managed session %s: cannot create client to poll: %v", sessionID, cause)
	out := buildOutput("error", sessionID)
	return ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out})
}

// handleTerminalSession emits the final messages for a terminal session and
// cleans up. While events are still being written it reschedules a poll, and
// while the result cannot be emitted it retries via polling — both bounded by
// maxPollAttempts so a stuck session cannot poll forever.
func (a *RunCloudAgent) handleTerminalSession(ctx core.ActionHookContext, client *runagent.Client, metadata *ExecutionMetadata, sess *runagent.ManagedSession, attempt, errs int) error {
	sessionID := metadata.Session.ID

	sm := a.fetchFinalMessages(ctx, client, sessionID)
	if (sm == nil || !sm.Complete) && attempt <= maxPollAttempts {
		// Events are not fully written yet (eventual consistency); retry.
		ctx.Logger.Warnf("Events not complete for session %s. Retrying poll.", sessionID)
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}

	// Emit the result (possibly partial if we ran past the retry budget).
	out := buildOutputFromSessionMessages(sess.Status, sessionID, sm)
	if err := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); err != nil {
		if attempt <= maxPollAttempts {
			// Do NOT delete the session: it holds the assembled result, so a
			// transient emit failure must be recoverable. Retry via polling.
			ctx.Logger.Warnf("Failed to emit result for session %s: %v. Retrying poll.", sessionID, err)
			return a.scheduleNextPoll(ctx, attempt+1, errs)
		}
		// Retry budget exhausted: reclaim the session and give up.
		cleanupManagedSessionFromHook(client, ctx, sessionID, false)
		return err
	}

	// Only persist terminal status after successful emit
	mergeSessionIntoMetadata(metadata, sess)
	_ = ctx.Metadata.Set(*metadata)

	cleanupManagedSessionFromHook(client, ctx, sessionID, false)
	return nil
}

// fetchFinalMessages returns the session's messages, or nil if they cannot be fetched.
func (a *RunCloudAgent) fetchFinalMessages(ctx core.ActionHookContext, client *runagent.Client, sessionID string) *runagent.SessionMessages {
	sm, err := client.GetSessionMessagesWithRetry(sessionID, finalMessageReads, finalMessageDelay)
	if err != nil {
		ctx.Logger.Warnf("Failed to fetch messages for session %s: %v.", sessionID, err)
		return nil
	}
	return sm
}

func (a *RunCloudAgent) scheduleNextPoll(ctx core.ActionHookContext, nextAttempt, errors int) error {
	interval := initialPoll * time.Duration(1<<uint(min(nextAttempt-1, 8)))
	if interval > maxPollInterval {
		interval = maxPollInterval
	}
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{
		"attempt": nextAttempt,
		"errors":  errors,
	}, interval)
}

func (a *RunCloudAgent) Cancel(ctx core.ExecutionContext) error {
	metadata := ExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil
	}
	if metadata.Session == nil || metadata.Session.ID == "" {
		return nil
	}
	client, err := runagent.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil
	}
	if err := client.SendManagedSessionInterrupt(metadata.Session.ID); err != nil {
		ctx.Logger.Warnf("Failed to interrupt managed session %s: %v", metadata.Session.ID, err)
	} else {
		ctx.Logger.Infof("Sent interrupt to managed session %s", metadata.Session.ID)
	}
	// Best effort cleanup; may fail if session is still running.
	_ = client.DeleteManagedSession(metadata.Session.ID)
	cleanupUploadedFiles(client, ctx, ctx.Logger.Warnf)
	cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
	return nil
}
