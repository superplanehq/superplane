package claude

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	integrationSecretAnthropicAPIKey    = "ANTHROPIC_API_KEY"
	integrationSecretAnthropicAuthToken = "ANTHROPIC_AUTH_TOKEN"
)

// oauthTokenPrefix identifies a Claude Code OAuth token as opposed to an
// Anthropic API key. OAuth tokens must reach the CLI as a bearer-style
// subscription token (ANTHROPIC_AUTH_TOKEN); API keys go to the CLI as
// ANTHROPIC_API_KEY. The prefix makes the two mutually exclusive, so a single
// "apiKey" field can accept either.
const oauthTokenPrefix = "sk-ant-oat"

func (i *Claude) ResolveSecrets(ctx core.IntegrationSecretContext) (map[string][]byte, error) {
	apiKey, err := ctx.Integration.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	// Claude Code OAuth tokens must be passed to the CLI as ANTHROPIC_AUTH_TOKEN;
	// an API key goes into ANTHROPIC_API_KEY. Handing an OAuth token to the
	// wrong variable makes the CLI reject it with "Invalid API key".
	secretName := integrationSecretAnthropicAPIKey
	if strings.HasPrefix(key, oauthTokenPrefix) {
		secretName = integrationSecretAnthropicAuthToken
	}

	return map[string][]byte{
		secretName: []byte(key),
	}, nil
}
