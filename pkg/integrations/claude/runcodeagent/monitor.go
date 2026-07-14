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
		// No session was recorded (anomalous): reclaim any stray resources and
		// finish as an error instead of leaving the node running indefinitely.
		ctx.Logger.Errorf("poll: execution metadata has no session id; finishing as error")
		if client, err := runagent.NewClient(ctx.HTTP, ctx.Integration); err == nil {
			a.teardown(client, meta, false, persistSession(ctx.Configuration), ctx.Logger.Warnf)
		}
		out := buildOutput("error", "", meta.Branch, nil, meta.PrURL)
		return ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out})
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

// emitFinal emits a terminal payload and, only after a successful emit, reclaims
// the provisioned resources. On an emit failure it returns the error WITHOUT
// tearing down, so the session survives for the hook to retry.
func (a *RunCodeAgent) emitFinal(ctx core.ActionHookContext, client *runagent.Client, meta *ExecutionMetadata, out OutputPayload, interrupt bool) error {
	if err := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); err != nil {
		ctx.Logger.Warnf("Failed to emit result for session %s: %v.", meta.Session.ID, err)
		return err
	}
	a.teardown(client, meta, interrupt, persistSession(ctx.Configuration), ctx.Logger.Warnf)
	return nil
}

// finishTimeout reports a timeout and reclaims the still-running session.
func (a *RunCodeAgent) finishTimeout(ctx core.ActionHookContext, client *runagent.Client, meta *ExecutionMetadata) error {
	ctx.Logger.Errorf("Session %s exceeded max poll attempts", meta.Session.ID)
	out := buildOutput("timeout", meta.Session.ID, meta.Branch, nil, meta.PrURL)
	return a.emitFinal(ctx, client, meta, out, true)
}

// handlePollError retries a failed status read, then reports an error and reclaims.
func (a *RunCodeAgent) handlePollError(ctx core.ActionHookContext, client *runagent.Client, meta *ExecutionMetadata, attempt, errs int) error {
	errs++
	if errs < maxPollErrors {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}
	ctx.Logger.Errorf("Session %s: polling failed repeatedly", meta.Session.ID)
	out := buildOutput("error", meta.Session.ID, meta.Branch, nil, meta.PrURL)
	return a.emitFinal(ctx, client, meta, out, true)
}

// handleClientError surfaces client/config failures as errors (not timeouts).
func (a *RunCodeAgent) handleClientError(ctx core.ActionHookContext, meta *ExecutionMetadata, attempt, errs int, cause error) error {
	errs++
	if errs < maxPollErrors {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}
	ctx.Logger.Errorf("Session %s: cannot create client to poll: %v", meta.Session.ID, cause)
	out := buildOutput("error", meta.Session.ID, meta.Branch, nil, meta.PrURL)
	if err := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); err != nil {
		return err
	}
	// Best-effort reclaim: retry building a client now that the run is finished.
	if client, cErr := runagent.NewClient(ctx.HTTP, ctx.Integration); cErr == nil {
		a.teardown(client, meta, true, persistSession(ctx.Configuration), ctx.Logger.Warnf)
	} else {
		ctx.Logger.Warnf("Cannot reclaim resources for session %s: client unavailable: %v", meta.Session.ID, cErr)
	}
	return nil
}

func (a *RunCodeAgent) handleTerminalSession(ctx core.ActionHookContext, client *runagent.Client, meta *ExecutionMetadata, sess *runagent.ManagedSession, attempt, errs int) error {
	sm, err := client.GetSessionMessagesWithRetry(meta.Session.ID, finalMessageReads, finalMessageDelay)
	if (err != nil || sm == nil || !sm.Complete) && attempt <= maxPollAttempts {
		// Give the event stream a chance to finish writing so we can assemble the
		// full result. Once the budget is spent, emit the session's real terminal
		// status (idle/terminated) with whatever we have — the API already
		// confirmed the session finished, so this is not a timeout.
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
	a.teardown(client, meta, false, persistSession(ctx.Configuration), ctx.Logger.Warnf)
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
	a.teardown(client, meta, true, persistSession(ctx.Configuration), ctx.Logger.Warnf)
	return nil
}
