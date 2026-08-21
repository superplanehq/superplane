package claude

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	integrationSecretAnthropicAPIKey = "ANTHROPIC_API_KEY"
	// Claude Code reads a subscription token from this variable. The same token
	// in ANTHROPIC_API_KEY is rejected as an invalid key.
	integrationSecretClaudeCodeOAuthToken = "CLAUDE_CODE_OAUTH_TOKEN"
)

func (i *Claude) ResolveSecrets(ctx core.IntegrationSecretContext) (map[string][]byte, error) {
	apiKey, err := ctx.Integration.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	if strings.HasPrefix(key, oauthTokenPrefix) {
		return map[string][]byte{
			integrationSecretClaudeCodeOAuthToken: []byte(key),
		}, nil
	}

	return map[string][]byte{
		integrationSecretAnthropicAPIKey: []byte(key),
	}, nil
}
