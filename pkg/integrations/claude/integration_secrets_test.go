package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestResolveSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		apiKey     string
		secretName string
		secret     string
	}{
		{
			name:       "API key is resolved as ANTHROPIC_API_KEY",
			apiKey:     "sk-ant-api03-abc",
			secretName: integrationSecretAnthropicAPIKey,
			secret:     "sk-ant-api03-abc",
		},
		{
			name:       "OAuth token is resolved as ANTHROPIC_AUTH_TOKEN",
			apiKey:     "sk-ant-oat01-abc",
			secretName: integrationSecretAnthropicAuthToken,
			secret:     "sk-ant-oat01-abc",
		},
		{
			name:       "whitespace around the key is trimmed",
			apiKey:     "  sk-ant-api03-abc  ",
			secretName: integrationSecretAnthropicAPIKey,
			secret:     "sk-ant-api03-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets, err := (&Claude{}).ResolveSecrets(core.IntegrationSecretContext{
				Integration: &contexts.IntegrationContext{
					Configuration: map[string]any{
						"apiKey": tt.apiKey,
					},
				},
			})
			require.NoError(t, err)
			assert.Equal(t, []byte(tt.secret), secrets[tt.secretName])
			assert.Len(t, secrets, 1)
		})
	}
}

func TestResolveSecretsRejectsEmptyKey(t *testing.T) {
	t.Parallel()

	secrets, err := (&Claude{}).ResolveSecrets(core.IntegrationSecretContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "   ",
			},
		},
	})
	require.Error(t, err)
	assert.Nil(t, secrets)
}
