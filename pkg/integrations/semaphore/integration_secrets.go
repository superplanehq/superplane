package semaphore

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const integrationSecretSemaphoreAPIToken = "SEMAPHORE_API_TOKEN"

func (s *Semaphore) ResolveSecrets(ctx core.IntegrationSecretContext) (map[string][]byte, error) {
	token, err := resolveAPIToken(ctx.Integration)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		integrationSecretSemaphoreAPIToken: []byte(token),
	}, nil
}

func resolveAPIToken(integrationCtx core.IntegrationContext) (string, error) {
	if integrationCtx.LegacySetup() {
		apiToken, err := integrationCtx.GetConfig("apiToken")
		if err != nil {
			return "", fmt.Errorf("failed to get API token: %w", err)
		}

		token := strings.TrimSpace(string(apiToken))
		if token == "" {
			return "", fmt.Errorf("API token is required")
		}

		return token, nil
	}

	token, err := integrationCtx.Secrets().Get(SecretAPIToken)
	if err != nil {
		return "", fmt.Errorf("failed to get API token: %w", err)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("API token is required")
	}

	return token, nil
}
