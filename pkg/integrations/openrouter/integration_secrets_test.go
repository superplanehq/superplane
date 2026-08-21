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
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "sk-or-test",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("sk-or-test"), secrets[integrationSecretOpenRouterAPIKey])
}
