package github

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func TestGitHub__ResolveSecrets__PAT(t *testing.T) {
	t.Parallel()

	integrationCtx := &contexts.IntegrationContext{
		NewSetupFlow: true,
		CurrentProperties: map[string]any{
			common.PropertyAuthMethod: common.AuthMethodPAT,
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			common.SecretPAT: {Name: common.SecretPAT, Value: []byte("ghp_test_token")},
		},
	}

	secrets, err := (&GitHub{}).ResolveSecrets(core.IntegrationSecretContext{
		HTTP:        &contexts.HTTPContext{},
		Integration: integrationCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("ghp_test_token"), secrets.Values[integrationSecretGitHubToken])
	assert.Contains(t, secrets.Usage, "GITHUB_TOKEN")
	assert.Contains(t, secrets.Usage, "The gh CLI is already installed")
	assert.Contains(t, secrets.Usage, "Do not download or install gh")
	assert.Contains(t, secrets.Usage, "https://github.com/<owner>/<repo>.git")
	assert.NotContains(t, secrets.Usage, "x-access-token")
	assert.NotContains(t, secrets.Usage, "ghp_test_token")
	assert.Equal(t, githubSetupName, secrets.SetupName)
	assert.Contains(t, secrets.Setup, "gh auth setup-git --hostname github.com --force")
	assert.NotContains(t, secrets.Setup, "x-access-token")
	assert.NotContains(t, secrets.Setup, "ghp_test_token")
}

func TestGitHub__ResolveSecrets__HostedBinding(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	organization, err := models.CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)
	t.Setenv(common.EnvGitHubAppID, "99")
	t.Setenv(common.EnvGitHubAppSlug, "superplane")
	t.Setenv(common.EnvGitHubAppPrivateKey, string(githubPrivateKeyPEM(t)))
	t.Setenv(common.EnvGitHubAppWebhookSecret, "whsec")

	require.NoError(t, models.UpsertVCSProviderInstallation(database.Conn(), &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 501,
		AccountLogin:   "acme",
	}))
	integration, err := models.FindOrCreateVCSProviderBinding(database.Conn(), organization.ID, models.ProviderGitHub, 501, "acme")
	require.NoError(t, err)

	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		mocks.GitHubResponse(http.StatusCreated, `{"token":"ghs_installation","expires_at":"2030-01-01T00:00:00Z"}`),
	}}
	integrationCtx := &contexts.IntegrationContext{
		IntegrationID: integration.ID.String(),
		Metadata:      common.Metadata{HostedApp: true},
	}

	secrets, err := (&GitHub{}).ResolveSecrets(core.IntegrationSecretContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, []byte("ghs_installation"), secrets.Values[integrationSecretGitHubToken])
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
	assert.Equal(t, "/app/installations/501/access_tokens", httpCtx.Requests[0].URL.Path)
}
