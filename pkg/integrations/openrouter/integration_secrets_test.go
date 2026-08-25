package openrouter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestResolveSecrets(t *testing.T) {
	t.Parallel()

	secrets, err := (&OpenRouter{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: connectedIntegration(map[string]any{}),
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("sk-or-v1-test"), secrets[integrationSecretOpenRouterAPIKey])
}

func TestResolveSecretsRequiresAPIKey(t *testing.T) {
	t.Parallel()

	_, err := (&OpenRouter{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OpenRouter API key is required")
}
