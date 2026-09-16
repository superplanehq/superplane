package claude

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	integrationSecretAnthropicAPIKey = "ANTHROPIC_API_KEY"
	claudeSecretUsage                = `An Anthropic API key is available in the ANTHROPIC_API_KEY environment variable.
Do not print the key.`
)

func (i *Claude) ResolveSecrets(ctx core.IntegrationSecretContext) (core.IntegrationSecrets, error) {
	apiKey, err := ctx.Integration.GetConfig("apiKey")
	if err != nil {
		return core.IntegrationSecrets{}, fmt.Errorf("failed to get API key: %w", err)
	}

	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return core.IntegrationSecrets{}, fmt.Errorf("apiKey is required")
	}

	return core.IntegrationSecrets{
		Values: map[string][]byte{
			integrationSecretAnthropicAPIKey: []byte(key),
		},
		Usage: claudeSecretUsage,
	}, nil
}
