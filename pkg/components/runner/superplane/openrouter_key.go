package superplane

import (
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	openrouterapi "github.com/superplanehq/superplane/pkg/integrations/openrouter"
	"github.com/superplanehq/superplane/pkg/models"
)

var openrouterNowUTC = func() time.Time {
	return time.Now().UTC()
}

func mintOpenRouterRunnerKey(ctx core.ExecutionContext, access core.HostedLLMAccess, executionTimeoutSeconds int) (core.HostedLLMAccess, error) {
	if strings.TrimSpace(access.ManagementKey) == "" {
		return access, fmt.Errorf("%w: %s", models.ErrHostedLLMProviderNoManagementKey, models.UsageProviderOpenRouter)
	}

	expiresAt := openrouterapi.ChildKeyExpiresAt(openrouterNowUTC(), executionTimeoutSeconds)
	client := openrouterapi.NewManagementClient(ctx.HTTP, access.ManagementKey)
	created, err := client.CreateKey(openrouterapi.CreateKeyRequest{
		Name:      openrouterapi.RunnerKeyName(ctx.RunID),
		ExpiresAt: openrouterapi.FormatKeyExpiresAt(expiresAt),
	})
	if err != nil {
		return access, fmt.Errorf("mint OpenRouter runner key: %w", err)
	}

	if ctx.ExecutionState != nil {
		if err := ctx.ExecutionState.SetKV(runner.OpenRouterChildKeyHashKV, created.Hash); err != nil {
			_ = client.DeleteKey(created.Hash)
			return access, fmt.Errorf("store OpenRouter key hash: %w", err)
		}
	}

	access.APIKey = created.Key
	return access, nil
}

func revokeMintedOpenRouterRunnerKey(ctx core.ExecutionContext) {
	runner.RevokeOpenRouterChildKey(ctx.HTTP, ctx.ExecutionState, ctx.HostedLLM, ctx.Logger)
}
