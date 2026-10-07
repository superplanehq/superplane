package cloudflare

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	integrationSecretCloudflareAPIToken  = "CLOUDFLARE_API_TOKEN"
	integrationSecretCloudflareAccountID = "CLOUDFLARE_ACCOUNT_ID"
	cloudflareSecretUsage                = `A Cloudflare API token is available in the CLOUDFLARE_API_TOKEN environment variable.
The account ID is available in the CLOUDFLARE_ACCOUNT_ID environment variable.
Use this token as a Bearer token for the Cloudflare API.
wrangler reads CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID.
Do not print the token.`
	cloudflareSecretUsageWithoutAccountID = `A Cloudflare API token is available in the CLOUDFLARE_API_TOKEN environment variable.
Use this token as a Bearer token for the Cloudflare API.
Do not print the token.`
)

func (c *Cloudflare) ResolveSecrets(ctx core.IntegrationSecretContext) (core.IntegrationSecrets, error) {
	apiToken, err := ctx.Integration.GetConfig("apiToken")
	if err != nil {
		return core.IntegrationSecrets{}, fmt.Errorf("failed to get API token: %w", err)
	}

	token := strings.TrimSpace(string(apiToken))
	if token == "" {
		return core.IntegrationSecrets{}, fmt.Errorf("apiToken is required")
	}

	values := map[string][]byte{
		integrationSecretCloudflareAPIToken: []byte(token),
	}

	accountID := accountIDFromIntegration(ctx.Integration)
	if accountID == "" {
		return core.IntegrationSecrets{
			Values: values,
			Usage:  cloudflareSecretUsageWithoutAccountID,
		}, nil
	}

	values[integrationSecretCloudflareAccountID] = []byte(accountID)
	return core.IntegrationSecrets{
		Values: values,
		Usage:  cloudflareSecretUsage,
	}, nil
}
