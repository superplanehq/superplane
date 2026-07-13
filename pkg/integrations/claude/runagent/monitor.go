package runagent

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// HandleWebhook — Managed Agents completion is observed via polling, not webhooks.
func (a *RunAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (a *RunAgent) Hooks() []core.Hook {
	return []core.Hook{{
		Name: "poll",
		Type: core.HookTypeInternal,
	}}
}

func (a *RunAgent) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == "poll" {
		return a.poll(ctx)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (a *RunAgent) poll(ctx core.ActionHookContext) error {
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

	attempt := 1
	errs := 0
	if a, ok := ctx.Parameters["attempt"].(float64); ok {
		attempt = int(a)
	}
	if e, ok := ctx.Parameters["errors"].(float64); ok {
		errs = int(e)
	}

	if attempt > maxPollAttempts {
		ctx.Logger.Errorf("Managed session %s exceeded max poll attempts", metadata.Session.ID)
		out := buildOutput("timeout", metadata.Session.ID)
		if emitErr := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); emitErr != nil {
			return emitErr
		}
		if c, cErr := NewClient(ctx.HTTP, ctx.Integration); cErr == nil {
			cleanupUploadedFilesFromHook(c, ctx, ctx.Logger.Warnf)
			cleanupManagedVaultFromHook(c, ctx, ctx.Logger.Warnf)
		}
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}

	sess, err := client.GetManagedSession(metadata.Session.ID)
	if err != nil {
		errs++
		if errs >= maxPollErrors {
			ctx.Logger.Errorf("Managed session %s: polling failed repeatedly: %v", metadata.Session.ID, err)
			out := buildOutput("error", metadata.Session.ID)
			if emitErr := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); emitErr != nil {
				return emitErr
			}
			cleanupUploadedFilesFromHook(client, ctx, ctx.Logger.Warnf)
			cleanupManagedVaultFromHook(client, ctx, ctx.Logger.Warnf)
			return nil
		}
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}

	// Don't write terminal status to metadata yet — we only persist it
	// after a successful emit to avoid blocking future poll retries.
	if sess == nil {
		return a.scheduleNextPoll(ctx, attempt+1, errs)
	}
	if isSessionTerminal(sess.Status) {
		sm, err := client.GetSessionMessagesWithRetry(metadata.Session.ID, finalMessageReads, finalMessageDelay)
		if err != nil {
			ctx.Logger.Warnf("Failed to fetch messages for session %s: %v. Retrying poll.", metadata.Session.ID, err)
			return a.scheduleNextPoll(ctx, attempt+1, errs+1)
		}
		if sm == nil || !sm.Complete {
			ctx.Logger.Warnf("Events not complete for session %s after retries. Retrying poll.", metadata.Session.ID)
			return a.scheduleNextPoll(ctx, attempt+1, errs)
		}

		out := buildOutputFromSessionMessages(sess.Status, metadata.Session.ID, sm)
		out.Artifacts = CollectSessionArtifacts(client, metadata.Session.ID, ctx.Logger.Warnf)
		if emitErr := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); emitErr != nil {
			return emitErr
		}

		// Only persist terminal status after successful emit
		mergeSessionIntoMetadata(&metadata, sess)
		_ = ctx.Metadata.Set(metadata)

		if err := client.DeleteManagedSession(metadata.Session.ID); err != nil {
			ctx.Logger.Warnf("Failed to delete managed session %s: %v", metadata.Session.ID, err)
		}
		cleanupUploadedFilesFromHook(client, ctx, ctx.Logger.Warnf)
		cleanupManagedVaultFromHook(client, ctx, ctx.Logger.Warnf)
		return nil
	}

	return a.scheduleNextPoll(ctx, attempt+1, 0)
}

func (a *RunAgent) scheduleNextPoll(ctx core.ActionHookContext, nextAttempt, errors int) error {
	interval := initialPoll * time.Duration(1<<uint(min(nextAttempt-1, 8)))
	if interval > maxPollInterval {
		interval = maxPollInterval
	}
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{
		"attempt": nextAttempt,
		"errors":  errors,
	}, interval)
}

func (a *RunAgent) Cancel(ctx core.ExecutionContext) error {
	metadata := ExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil
	}
	if metadata.Session == nil || metadata.Session.ID == "" {
		return nil
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
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
