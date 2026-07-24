package openai

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// Env-style keys under which OpenAI credentials are exported to consumers
// (runners, components, etc.).
//
// The organization admin key is intentionally never exported here. It is a
// highly privileged credential that can read organization-wide usage and
// costs, and it is only consumed internally by the Get Usage action (which
// reads it directly from the integration config). Exposing it to runners and
// components would broaden its blast radius unnecessarily.
const (
	integrationSecretOpenAIAPIKey  = "OPENAI_API_KEY"
	integrationSecretOpenAIBaseURL = "OPENAI_BASE_URL"
)

// ResolveSecrets implements core.IntegrationSecretProvider, materializing the
// OpenAI API key, plus the base URL when it is configured.
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

	// baseURL is optional; only export it when set.
	if baseURLBytes, err := ctx.Integration.GetConfig("baseURL"); err == nil {
		if baseURL := strings.TrimSpace(string(baseURLBytes)); baseURL != "" {
			secrets[integrationSecretOpenAIBaseURL] = []byte(baseURL)
		}
	}

	return secrets, nil
}
