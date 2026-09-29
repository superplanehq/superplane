package cloudflare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestResolveSecrets(t *testing.T) {
	t.Parallel()

	secrets, err := (&Cloudflare{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":  "cf-token",
				"accountId": "acc-123",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("cf-token"), secrets.Values[integrationSecretCloudflareAPIToken])
	assert.Equal(t, []byte("acc-123"), secrets.Values[integrationSecretCloudflareAccountID])
	assert.Contains(t, secrets.Usage, "CLOUDFLARE_API_TOKEN")
	assert.Contains(t, secrets.Usage, "CLOUDFLARE_ACCOUNT_ID")
	assert.NotContains(t, secrets.Usage, "cf-token")
	assert.Empty(t, secrets.Setup)
}

func TestResolveSecretsWithoutAccountID(t *testing.T) {
	t.Parallel()

	secrets, err := (&Cloudflare{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "cf-token",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("cf-token"), secrets.Values[integrationSecretCloudflareAPIToken])
	assert.Empty(t, secrets.Values[integrationSecretCloudflareAccountID])
	assert.Contains(t, secrets.Usage, "CLOUDFLARE_API_TOKEN")
	assert.NotContains(t, secrets.Usage, "CLOUDFLARE_ACCOUNT_ID")
	assert.NotContains(t, secrets.Usage, "cf-token")
	assert.Empty(t, secrets.Setup)
}

func TestResolveSecretsUsesAccountIDFromMetadata(t *testing.T) {
	t.Parallel()

	secrets, err := (&Cloudflare{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "cf-token",
			},
			Metadata: Metadata{AccountID: "meta-acc"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("meta-acc"), secrets.Values[integrationSecretCloudflareAccountID])
}

func TestResolveSecretsRequiresAPIToken(t *testing.T) {
	t.Parallel()

	_, err := (&Cloudflare{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "  ",
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiToken is required")
}
