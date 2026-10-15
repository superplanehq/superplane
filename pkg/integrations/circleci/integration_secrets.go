package circleci

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	integrationSecretCircleCIAPIToken = "CIRCLECI_API_TOKEN"
	circleCISecretUsage               = `A CircleCI API token is available in the CIRCLECI_API_TOKEN environment variable.
Use this token with the Circle-Token HTTP header to access CircleCI build details and logs.
Do not print the token.`
)

func (c *CircleCI) ResolveSecrets(ctx core.IntegrationSecretContext) (core.IntegrationSecrets, error) {
	apiToken, err := ctx.Integration.GetConfig("apiToken")
	if err != nil {
		return core.IntegrationSecrets{}, fmt.Errorf("failed to get API token: %w", err)
	}

	token := strings.TrimSpace(string(apiToken))
	if token == "" {
		return core.IntegrationSecrets{}, fmt.Errorf("apiToken is required")
	}

	return core.IntegrationSecrets{
		Values: map[string][]byte{
			integrationSecretCircleCIAPIToken: []byte(token),
		},
		Usage: circleCISecretUsage,
	}, nil
}
