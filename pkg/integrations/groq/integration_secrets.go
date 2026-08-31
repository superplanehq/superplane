package groq

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const integrationSecretGroqAPIKey = "GROQ_API_KEY"

func (g *Groq) ResolveSecrets(ctx core.IntegrationSecretContext) (map[string][]byte, error) {
	key, err := integrationAPIKey(ctx.Integration)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{integrationSecretGroqAPIKey: []byte(key)}, nil
}

func integrationAPIKey(integration core.IntegrationContext) (string, error) {
	if integration == nil {
		return "", fmt.Errorf("no integration context")
	}

	apiKey, err := integration.GetConfig("apiKey")
	if err != nil {
		return "", fmt.Errorf("failed to get Groq API key: %w", err)
	}

	return normalizeAPIKey(string(apiKey))
}

func configurationAPIKey(configuration any) (string, error) {
	config := Configuration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return "", fmt.Errorf("failed to decode Groq configuration: %w", err)
	}

	return normalizeAPIKey(config.APIKey)
}

func normalizeAPIKey(apiKey string) (string, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return "", fmt.Errorf("apiKey is required")
	}

	return key, nil
}
