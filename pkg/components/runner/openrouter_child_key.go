package runner

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/openrouter"
	"github.com/superplanehq/superplane/pkg/models"
)

const OpenRouterChildKeyHashKV = "openrouter_key_hash"

func revokeOpenRouterChildKey(httpCtx core.HTTPContext, state core.ExecutionStateContext, hosted core.HostedLLMContext, logger *log.Entry) {
	if httpCtx == nil || state == nil {
		return
	}

	hash, err := state.GetKV(OpenRouterChildKeyHashKV)
	if err != nil {
		return
	}
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return
	}
	if hosted == nil {
		if logger != nil {
			logger.Warn("runner: cannot delete OpenRouter child key: hosted credentials are not available")
		}
		return
	}

	access, err := hosted.Resolve(models.UsageProviderOpenRouter)
	if err != nil {
		if logger != nil {
			logger.WithError(err).Warn("runner: failed to resolve OpenRouter management key for child key delete")
		}
		return
	}
	if strings.TrimSpace(access.ManagementKey) == "" {
		if logger != nil {
			logger.Warn("runner: cannot delete OpenRouter child key: provisioning API key is missing")
		}
		return
	}

	client := openrouter.NewManagementClient(httpCtx, access.ManagementKey)
	if err := client.DeleteKey(hash); err != nil && logger != nil {
		logger.WithError(err).Warn("runner: failed to delete OpenRouter child key")
	}
}
