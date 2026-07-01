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

	if attempt > maxPollAttempts {
		return a.finishTimeout(ctx, sessionID)
	}

	client, err := runagent.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}

	sess, err := client.GetManagedSession(sessionID)
	if err != nil {
		return a.handlePollError(ctx, client, sessionID, attempt, errs)
	}

	// Don't write terminal status to metadata yet — we only persist it
	// after a successful emit to avoid blocking future poll retries.
	if sess == nil {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}
	if !isSessionTerminal(sess.Status) {
		return a.scheduleNextPoll(ctx, attempt+1, 0)
	}
	return a.handleTerminalSession(ctx, client, &metadata, sess, attempt, errs)
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

// finishTimeout emits a timeout payload and cleans up best-effort.
func (a *RunCloudAgent) finishTimeout(ctx core.ActionHookContext, sessionID string) error {
	ctx.Logger.Errorf("Managed session %s exceeded max poll attempts", sessionID)
	out := buildOutput("timeout", sessionID)
	if err := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); err != nil {
		return err
	}
	if c, cErr := runagent.NewClient(ctx.HTTP, ctx.Integration); cErr == nil {
		cleanupUploadedFilesFromHook(c, ctx, ctx.Logger.Warnf)
		cleanupManagedVaultFromHook(c, ctx, ctx.Logger.Warnf)
	}
	return nil
}

// handlePollError records a failed status read and emits an error payload once
// the retry budget is exhausted, otherwise schedules another poll.
func (a *RunCloudAgent) handlePollError(ctx core.ActionHookContext, client *runagent.Client, sessionID string, attempt, errs int) error {
	errs++
	if errs < maxPollErrors {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}

	ctx.Logger.Errorf("Managed session %s: polling failed repeatedly", sessionID)
	out := buildOutput("error", sessionID)
	if err := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); err != nil {
		return err
	}
	cleanupUploadedFilesFromHook(client, ctx, ctx.Logger.Warnf)
	cleanupManagedVaultFromHook(client, ctx, ctx.Logger.Warnf)
	return nil
}

// handleTerminalSession emits the final messages for a terminal session and
// cleans up, or reschedules a poll if the events are not yet fully written.
func (a *RunCloudAgent) handleTerminalSession(ctx core.ActionHookContext, client *runagent.Client, metadata *ExecutionMetadata, sess *runagent.ManagedSession, attempt, errs int) error {
	sessionID := metadata.Session.ID

	sm, err := client.GetSessionMessagesWithRetry(sessionID, finalMessageReads, finalMessageDelay)
	if err != nil {
		ctx.Logger.Warnf("Failed to fetch messages for session %s: %v. Retrying poll.", sessionID, err)
		return a.scheduleNextPoll(ctx, attempt+1, errs+1)
	}
	if sm == nil || !sm.Complete {
		ctx.Logger.Warnf("Events not complete for session %s after retries. Retrying poll.", sessionID)
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}

	out := buildOutputFromSessionMessages(sess.Status, sessionID, sm)
	if err := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); err != nil {
		return err
	}

	// Only persist terminal status after successful emit
	mergeSessionIntoMetadata(metadata, sess)
	_ = ctx.Metadata.Set(*metadata)

	if err := client.DeleteManagedSession(sessionID); err != nil {
		ctx.Logger.Warnf("Failed to delete managed session %s: %v", sessionID, err)
	}
	cleanupUploadedFilesFromHook(client, ctx, ctx.Logger.Warnf)
	cleanupManagedVaultFromHook(client, ctx, ctx.Logger.Warnf)
	return nil
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
