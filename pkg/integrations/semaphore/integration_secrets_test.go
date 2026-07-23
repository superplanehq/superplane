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
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("sem-token"), secrets[integrationSecretSemaphoreAPIToken])
}
