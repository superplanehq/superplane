package openrouter

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const integrationSecretOpenRouterAPIKey = "OPENROUTER_API_KEY"

func (o *OpenRouter) ResolveSecrets(ctx core.IntegrationSecretContext) (map[string][]byte, error) {
	apiKey, err := ctx.Integration.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	return map[string][]byte{
		integrationSecretOpenRouterAPIKey: []byte(key),
	}, nil
}
