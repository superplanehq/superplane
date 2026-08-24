package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestResolveSecrets(t *testing.T) {
	t.Parallel()

	t.Run("personal access token", func(t *testing.T) {
		t.Parallel()

		secrets, err := (&GitLab{}).ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "glpat-test",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, []byte("glpat-test"), secrets[integrationSecretGitLabToken])
	})

	t.Run("app OAuth", func(t *testing.T) {
		t.Parallel()

		secrets, err := (&GitLab{}).ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType": AuthTypeAppOAuth,
				},
				CurrentSecrets: map[string]core.IntegrationSecret{
					OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("oauth-test")},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, []byte("oauth-test"), secrets[integrationSecretGitLabToken])
	})

	t.Run("missing authType", func(t *testing.T) {
		t.Parallel()

		_, err := (&GitLab{}).ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{},
			},
		})
		require.Error(t, err)
	})

	t.Run("missing personal access token", func(t *testing.T) {
		t.Parallel()

		_, err := (&GitLab{}).ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "",
				},
			},
		})
		require.Error(t, err)
	})

	t.Run("missing OAuth access token", func(t *testing.T) {
		t.Parallel()

		_, err := (&GitLab{}).ResolveSecrets(core.IntegrationSecretContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType": AuthTypeAppOAuth,
				},
			},
		})
		require.Error(t, err)
	})
}
