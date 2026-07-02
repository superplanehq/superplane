package runcodeagent

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runagent"
)

// HandleWebhook — Managed Agents completion is observed via polling, not webhooks.
func (a *RunCodeAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (a *RunCodeAgent) Hooks() []core.Hook {
	return []core.Hook{{Name: "poll", Type: core.HookTypeInternal}}
}

func (a *RunCodeAgent) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == "poll" {
		return a.poll(ctx)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (a *RunCodeAgent) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	meta := &ExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), meta); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}
	if meta.Session == nil || meta.Session.ID == "" {
		return nil
	}
	attempt, errs := parsePollParams(ctx)

	client, err := runagent.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return a.handleClientError(ctx, meta, attempt, errs, err)
	}

	sess, err := client.GetManagedSession(meta.Session.ID)
	if err != nil {
		return a.handlePollError(ctx, client, meta, attempt, errs)
	}

	if sess != nil && isSessionTerminal(sess.Status) {
		return a.handleTerminalSession(ctx, client, meta, sess, attempt, errs)
	}

	if attempt > maxPollAttempts {
		return a.finishTimeout(ctx, client, meta)
	}
	return a.scheduleNextPoll(ctx, attempt+1, 0)
}

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

// finishTimeout reclaims the still-running session and emits a timeout. Cleanup
// runs before the emit so nothing leaks even if the emit fails.
func (a *RunCodeAgent) finishTimeout(ctx core.ActionHookContext, client *runagent.Client, meta *ExecutionMetadata) error {
	ctx.Logger.Errorf("Session %s exceeded max poll attempts", meta.Session.ID)
	a.teardown(client, meta, true, ctx.Logger.Warnf)
	out := buildOutput("timeout", meta.Session.ID, meta.Branch, nil, meta.PrURL)
	return ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out})
}

// handlePollError retries a failed status read, then reclaims + reports an error.
func (a *RunCodeAgent) handlePollError(ctx core.ActionHookContext, client *runagent.Client, meta *ExecutionMetadata, attempt, errs int) error {
	errs++
	if errs < maxPollErrors {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}
	ctx.Logger.Errorf("Session %s: polling failed repeatedly", meta.Session.ID)
	a.teardown(client, meta, true, ctx.Logger.Warnf)
	out := buildOutput("error", meta.Session.ID, meta.Branch, nil, meta.PrURL)
	return ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out})
}

// handleClientError surfaces client/config failures as errors (not timeouts).
func (a *RunCodeAgent) handleClientError(ctx core.ActionHookContext, meta *ExecutionMetadata, attempt, errs int, cause error) error {
	errs++
	if errs < maxPollErrors {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}
	ctx.Logger.Errorf("Session %s: cannot create client to poll: %v", meta.Session.ID, cause)
	// Attempt teardown with a fresh client so the session/environment/vault/agent
	// are not left provisioned; if the client still can't be built there is
	// nothing more we can do via the API.
	if client, cErr := runagent.NewClient(ctx.HTTP, ctx.Integration); cErr == nil {
		a.teardown(client, meta, true, ctx.Logger.Warnf)
	} else {
		ctx.Logger.Warnf("Cannot reclaim resources for session %s: client unavailable: %v", meta.Session.ID, cErr)
	}
	out := buildOutput("error", meta.Session.ID, meta.Branch, nil, meta.PrURL)
	return ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out})
}

func (a *RunCodeAgent) handleTerminalSession(ctx core.ActionHookContext, client *runagent.Client, meta *ExecutionMetadata, sess *runagent.ManagedSession, attempt, errs int) error {
	sm, err := client.GetSessionMessagesWithRetry(meta.Session.ID, finalMessageReads, finalMessageDelay)
	if (err != nil || sm == nil || !sm.Complete) && attempt <= maxPollAttempts {
		ctx.Logger.Warnf("Events not complete for session %s. Retrying poll.", meta.Session.ID)
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}

	out := buildOutput(sess.Status, meta.Session.ID, meta.Branch, sm, meta.PrURL)
	if err := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); err != nil {
		// Never tear down on an emit failure: the session still holds the agent
		// output (PR URL, summary), so a transient emit failure must remain
		// recoverable. Retry via polling while within budget; past it, surface
		// the error without destroying the session.
		ctx.Logger.Warnf("Failed to emit result for session %s: %v.", meta.Session.ID, err)
		if attempt <= maxPollAttempts {
			return a.scheduleNextPoll(ctx, attempt+1, errs)
		}
		return err
	}

	mergeSessionIntoMetadata(meta, sess)
	_ = ctx.Metadata.Set(*meta)
	a.teardown(client, meta, false, ctx.Logger.Warnf)
	return nil
}

func (a *RunCodeAgent) scheduleNextPoll(ctx core.ActionHookContext, nextAttempt, errors int) error {
	interval := initialPoll * time.Duration(1<<uint(min(nextAttempt-1, 8)))
	if interval > maxPollInterval {
		interval = maxPollInterval
	}
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": nextAttempt, "errors": errors}, interval)
}

func (a *RunCodeAgent) Cancel(ctx core.ExecutionContext) error {
	meta := &ExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), meta); err != nil {
		return nil
	}
	client, err := runagent.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Warnf("Cancel: cannot build client to reclaim managed agent resources: %v", err)
		return nil
	}
	a.teardown(client, meta, true, ctx.Logger.Warnf)
	return nil
}
