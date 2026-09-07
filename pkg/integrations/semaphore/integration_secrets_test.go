package semaphore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestResolveSecrets(t *testing.T) {
	t.Parallel()

	secrets, err := (&Semaphore{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{
			NewSetupFlow: true,
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretAPIToken: {Name: SecretAPIToken, Value: []byte("sem-token")},
			},
			CurrentProperties: map[string]any{
				PropertyOrganizationURL: "https://acme.semaphoreci.com",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("sem-token"), secrets.Values[integrationSecretSemaphoreAPIToken])
	assert.Equal(t, []byte("https://acme.semaphoreci.com"), secrets.Values[integrationSecretSemaphoreOrganizationURL])
	assert.Contains(t, secrets.Usage, "SEMAPHORE_API_TOKEN")
	assert.Contains(t, secrets.Usage, "sem-ai")
	assert.NotContains(t, secrets.Usage, "sem-token")
	assert.Equal(t, semaphoreSetupName, secrets.SetupName)
	assert.Contains(t, secrets.Setup, "sem-ai")
	assert.Contains(t, secrets.Setup, "SEMAPHORE_API_TOKEN")
	assert.NotContains(t, secrets.Setup, "sem-token")
}

func TestResolveSecretsWithoutOrganizationURL(t *testing.T) {
	t.Parallel()

	secrets, err := (&Semaphore{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{
			NewSetupFlow: true,
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretAPIToken: {Name: SecretAPIToken, Value: []byte("sem-token")},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("sem-token"), secrets.Values[integrationSecretSemaphoreAPIToken])
	assert.Empty(t, secrets.Values[integrationSecretSemaphoreOrganizationURL])
	assert.Contains(t, secrets.Usage, "SEMAPHORE_API_TOKEN")
	assert.NotContains(t, secrets.Usage, "sem-ai")
	assert.Empty(t, secrets.Setup)
}
