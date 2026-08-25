package groq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestGroqResolveSecrets(t *testing.T) {
	secrets, err := (&Groq{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "  gsk-test  "},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, []byte("gsk-test"), secrets[integrationSecretGroqAPIKey])
}

func TestGroqResolveSecretsRequiresAPIKey(t *testing.T) {
	_, err := (&Groq{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "  "}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiKey is required")
}
