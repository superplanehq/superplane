package gitlab

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// integrationSecretGitLabToken is the env-style key under which the GitLab
// access token is exported to consumers (runners, components, etc.).
const integrationSecretGitLabToken = "GITLAB_TOKEN"

// ResolveSecrets implements core.IntegrationSecretProvider, materializing the
// GitLab access token for both personal-access-token and app-OAuth auth types.
func (g *GitLab) ResolveSecrets(ctx core.IntegrationSecretContext) (map[string][]byte, error) {
	authTypeBytes, err := ctx.Integration.GetConfig("authType")
	if err != nil {
		return nil, fmt.Errorf("failed to get authType: %w", err)
	}

	authType := strings.TrimSpace(string(authTypeBytes))
	if authType == "" {
		return nil, fmt.Errorf("authType is required")
	}

	token, err := getAuthToken(ctx.Integration, authType)
	if err != nil {
		return nil, err
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("access token is required")
	}

	return map[string][]byte{
		integrationSecretGitLabToken: []byte(token),
	}, nil
}
