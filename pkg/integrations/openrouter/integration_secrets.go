package openrouter

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const integrationSecretOpenRouterAPIKey = "OPENROUTER_API_KEY"

func (o *OpenRouter) ResolveSecrets(ctx core.IntegrationSecretContext) (map[string][]byte, error) {
	apiKey, err := findSecret(ctx.Integration, SecretAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenRouter API key: %w", err)
	}

	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, fmt.Errorf("OpenRouter API key is required")
	}

	return map[string][]byte{
		integrationSecretOpenRouterAPIKey: []byte(key),
	}, nil
}
