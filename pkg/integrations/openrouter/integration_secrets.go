package openrouter

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	integrationSecretOpenRouterAPIKey = "OPENROUTER_API_KEY"
	openRouterSecretUsage             = `An OpenRouter API key is available in the OPENROUTER_API_KEY environment variable.
Do not print the key.`
)

func (o *OpenRouter) ResolveSecrets(ctx core.IntegrationSecretContext) (core.IntegrationSecrets, error) {
	apiKey, err := findSecret(ctx.Integration, SecretAPIKey)
	if err != nil {
		return core.IntegrationSecrets{}, fmt.Errorf("failed to read OpenRouter API key: %w", err)
	}

	key := strings.TrimSpace(apiKey)
	if key == "" {
		return core.IntegrationSecrets{}, fmt.Errorf("OpenRouter API key is required")
	}

	return core.IntegrationSecrets{
		Values: map[string][]byte{
			integrationSecretOpenRouterAPIKey: []byte(key),
		},
		Usage: openRouterSecretUsage,
	}, nil
}
