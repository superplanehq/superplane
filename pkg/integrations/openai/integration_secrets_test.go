package openai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestResolveSecrets(t *testing.T) {
	t.Parallel()

	t.Run("api key only", func(t *testing.T) {
		t.Parallel()

		secrets, err := (&OpenAI{}).ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "sk-test",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, []byte("sk-test"), secrets[integrationSecretOpenAIAPIKey])
		assert.NotContains(t, secrets, integrationSecretOpenAIBaseURL)
	})

	t.Run("api key and base URL", func(t *testing.T) {
		t.Parallel()

		secrets, err := (&OpenAI{}).ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey":  "sk-test",
					"baseURL": "https://example.com/v1",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, []byte("sk-test"), secrets[integrationSecretOpenAIAPIKey])
		assert.Equal(t, []byte("https://example.com/v1"), secrets[integrationSecretOpenAIBaseURL])
	})

	t.Run("admin key is never exported", func(t *testing.T) {
		t.Parallel()

		secrets, err := (&OpenAI{}).ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey":   "sk-test",
					"adminKey": "sk-admin-test",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, []byte("sk-test"), secrets[integrationSecretOpenAIAPIKey])
		for key, value := range secrets {
			assert.NotEqual(t, "OPENAI_ADMIN_KEY", key)
			assert.NotEqual(t, []byte("sk-admin-test"), value)
		}
	})

	t.Run("missing api key", func(t *testing.T) {
		t.Parallel()

		_, err := (&OpenAI{}).ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "",
				},
			},
		})
		require.Error(t, err)
	})
}
