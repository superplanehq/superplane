package runner

import (
	"errors"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/openrouter"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	OpenRouterChildKeyHashKV = "openrouter_key_hash"

	childKeyDeleteMaxAttempts = 3
)

var childKeyDeleteRetryBackoff = 200 * time.Millisecond

func revokeOpenRouterChildKey(httpCtx core.HTTPContext, state core.ExecutionStateContext, hosted core.HostedLLMContext, logger *log.Entry) error {
	if httpCtx == nil || state == nil {
		return nil
	}

	hash, err := state.GetKV(OpenRouterChildKeyHashKV)
	if err != nil {
		if errors.Is(err, core.ErrExecutionKVNotFound) {
			return nil
		}
		if logger != nil {
			logger.WithError(err).Warn("runner: failed to read OpenRouter child key hash")
		}
		return err
	}
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return nil
	}
	if hosted == nil {
		if logger != nil {
			logger.Warn("runner: cannot delete OpenRouter child key: hosted credentials are not available")
		}
		return nil
	}

	access, err := hosted.Resolve(models.UsageProviderOpenRouter)
	if err != nil {
		if logger != nil {
			logger.WithError(err).Warn("runner: failed to resolve OpenRouter management key for child key delete")
		}
		return err
	}
	if strings.TrimSpace(access.ManagementKey) == "" {
		if logger != nil {
			logger.Warn("runner: cannot delete OpenRouter child key: provisioning API key is missing")
		}
		return nil
	}

	client := openrouter.NewManagementClient(httpCtx, access.ManagementKey)
	if err := deleteOpenRouterChildKey(client, hash); err != nil {
		if logger != nil {
			logger.WithError(err).Warn("runner: failed to delete OpenRouter child key")
		}
		return err
	}

	if err := state.SetKV(OpenRouterChildKeyHashKV, ""); err != nil && logger != nil {
		logger.WithError(err).Warn("runner: failed to clear OpenRouter child key hash")
	}
	return nil
}

func deleteOpenRouterChildKey(client *openrouter.Client, hash string) error {
	var lastErr error
	for attempt := range childKeyDeleteMaxAttempts {
		if attempt > 0 {
			time.Sleep(childKeyDeleteRetryBackoff)
		}
		lastErr = client.DeleteKey(hash)
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

func revokeOpenRouterChildKeyOrReschedule(ctx core.ActionHookContext) {
	err := revokeOpenRouterChildKey(ctx.HTTP, ctx.ExecutionState, ctx.HostedLLM, ctx.Logger)
	if err == nil || ctx.Requests == nil {
		return
	}

	taskID, _ := ctx.Parameters["task_id"].(string)
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return
	}

	organizationID, _ := ctx.Parameters["organization_id"].(string)
	if schedErr := ctx.Requests.ScheduleActionCall(hookActionPoll, map[string]any{
		"task_id":         taskID,
		"organization_id": organizationID,
	}, pollInterval); schedErr != nil && ctx.Logger != nil {
		ctx.Logger.WithError(schedErr).Warn("runner: failed to schedule OpenRouter child key delete retry")
	}
}
