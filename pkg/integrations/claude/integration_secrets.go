package claude

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const integrationSecretAnthropicAPIKey = "ANTHROPIC_API_KEY"

func (i *Claude) ResolveSecrets(ctx core.IntegrationSecretContext) (map[string][]byte, error) {
	apiKey, err := ctx.Integration.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	return map[string][]byte{
		integrationSecretAnthropicAPIKey: []byte(key),
	}, nil
}
