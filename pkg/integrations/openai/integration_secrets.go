package openai

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// Env-style keys under which OpenAI credentials are exported to consumers
// (runners, components, etc.).
const (
	integrationSecretOpenAIAPIKey   = "OPENAI_API_KEY"
	integrationSecretOpenAIAdminKey = "OPENAI_ADMIN_KEY"
	integrationSecretOpenAIBaseURL  = "OPENAI_BASE_URL"
)

// ResolveSecrets implements core.IntegrationSecretProvider, materializing the
// OpenAI API key, plus the admin key and base URL when they are configured.
func (o *OpenAI) ResolveSecrets(ctx core.IntegrationSecretContext) (map[string][]byte, error) {
	apiKeyBytes, err := ctx.Integration.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	apiKey := strings.TrimSpace(string(apiKeyBytes))
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	secrets := map[string][]byte{
		integrationSecretOpenAIAPIKey: []byte(apiKey),
	}

	// adminKey and baseURL are optional; only export them when set.
	if adminKeyBytes, err := ctx.Integration.GetConfig("adminKey"); err == nil {
		if adminKey := strings.TrimSpace(string(adminKeyBytes)); adminKey != "" {
			secrets[integrationSecretOpenAIAdminKey] = []byte(adminKey)
		}
	}

	if baseURLBytes, err := ctx.Integration.GetConfig("baseURL"); err == nil {
		if baseURL := strings.TrimSpace(string(baseURLBytes)); baseURL != "" {
			secrets[integrationSecretOpenAIBaseURL] = []byte(baseURL)
		}
	}

	return secrets, nil
}
