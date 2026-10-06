package bitbucket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Bitbucket__ResolveSecrets(t *testing.T) {
	b := &Bitbucket{}

	t.Run("workspace access token uses the x-token-auth git username", func(t *testing.T) {
		secrets, err := b.ResolveSecrets(core.IntegrationSecretContext{
			HTTP: &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"token": "workspace-token"},
				Metadata:      Metadata{AuthType: AuthTypeWorkspaceAccessToken},
			},
		})
		require.NoError(t, err)

		assert.Equal(t, map[string][]byte{
			"BITBUCKET_TOKEN":        []byte("workspace-token"),
			"BITBUCKET_GIT_USERNAME": []byte("x-token-auth"),
		}, secrets.Values)
		assert.Contains(t, secrets.Setup, "credential.https://bitbucket.org.helper")
		assert.NotContains(t, secrets.Setup, "workspace-token")
		assert.Equal(t, "Set up Bitbucket", secrets.SetupName)
	})

	t.Run("API token exports the email and the API token git username", func(t *testing.T) {
		secrets, err := b.ResolveSecrets(core.IntegrationSecretContext{
			HTTP: &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"token": "api-token", "email": "dev@example.com"},
				Metadata:      Metadata{AuthType: AuthTypeAPIToken},
			},
		})
		require.NoError(t, err)

		assert.Equal(t, map[string][]byte{
			"BITBUCKET_TOKEN":        []byte("api-token"),
			"BITBUCKET_GIT_USERNAME": []byte("x-bitbucket-api-token-auth"),
			"BITBUCKET_EMAIL":        []byte("dev@example.com"),
		}, secrets.Values)
	})

	t.Run("missing token returns an error", func(t *testing.T) {
		_, err := b.ResolveSecrets(core.IntegrationSecretContext{
			HTTP: &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"token": "  "},
				Metadata:      Metadata{AuthType: AuthTypeWorkspaceAccessToken},
			},
		})
		require.ErrorContains(t, err, "token is required")
	})
}
