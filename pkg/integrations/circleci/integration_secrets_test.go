package circleci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestResolveSecrets(t *testing.T) {
	provider, ok := any(&CircleCI{}).(core.IntegrationSecretProvider)
	require.True(t, ok, "CircleCI must provide credentials to runners")

	t.Run("exports configured token with agent guidance", func(t *testing.T) {
		secrets, err := provider.ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "  circleci-test-token\n"},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, map[string][]byte{"CIRCLECI_API_TOKEN": []byte("circleci-test-token")}, secrets.Values)
		assert.Contains(t, secrets.Usage, "CIRCLECI_API_TOKEN")
		assert.Contains(t, secrets.Usage, "Circle-Token")
		assert.NotContains(t, secrets.Usage, "circleci-test-token")
	})

	for name, token := range map[string]string{"empty": "", "whitespace": " \n\t"} {
		t.Run("rejects "+name+" token", func(t *testing.T) {
			secrets, err := provider.ResolveSecrets(core.IntegrationSecretContext{
				Integration: &contexts.IntegrationContext{
					Configuration: map[string]any{"apiToken": token},
				},
			})
			require.ErrorContains(t, err, "apiToken is required")
			assert.Empty(t, secrets.Values)
		})
	}

	t.Run("reports missing configuration", func(t *testing.T) {
		secrets, err := provider.ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{}},
		})
		require.ErrorContains(t, err, "failed to get API token")
		assert.Empty(t, secrets.Values)
	})
}
