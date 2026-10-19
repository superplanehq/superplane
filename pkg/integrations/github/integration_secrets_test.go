package github

import (
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
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
	assert.NotContains(t, secrets.Values, integrationSecretGitHubContextToken)
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
	require.NoError(t, models.ReplaceVCSProviderRepositories(database.Conn(), models.ProviderGitHub, 501, []models.VCSProviderRepository{
		{RepositoryID: 601, FullName: "acme/api"},
		{RepositoryID: 602, FullName: "acme/web"},
	}))
	require.NoError(t, models.GrantVCSProviderBindingRepository(database.Conn(), integration.ID, models.ProviderGitHub, 601))

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
	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"repository_ids":[601]}`, string(body))
	assert.NotContains(t, secrets.Values, integrationSecretGitHubContextToken)
}

func TestGitHub__ResolveSecrets__HostedContextRepositoriesAreReadOnly(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	organization, err := models.CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)
	other, err := models.CreateOrganization("org-"+uuid.NewString(), "")
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
	otherIntegration, err := models.FindOrCreateVCSProviderBinding(database.Conn(), other.ID, models.ProviderGitHub, 501, "acme")
	require.NoError(t, err)
	require.NoError(t, models.ReplaceVCSProviderRepositories(database.Conn(), models.ProviderGitHub, 501, []models.VCSProviderRepository{
		{RepositoryID: 601, FullName: "acme/api"},
		{RepositoryID: 602, FullName: "acme/web"},
		{RepositoryID: 603, FullName: "acme/secret"},
		{RepositoryID: 604, FullName: "acme/other"},
		{RepositoryID: 605, FullName: "acme/bb"},
	}))
	require.NoError(t, models.GrantVCSProviderBindingRepository(database.Conn(), integration.ID, models.ProviderGitHub, 601))

	factory := createContextRepoWorkspace(t, organization.ID, integration.ID)
	_, err = factory.CreateAgentResource(database.Conn(), models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/web",
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(database.Conn(), models.FactoryAgentResourceKindContextRepo, "", false, models.FactoryAgentResourceConfig{
		Repository: "acme/secret",
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(database.Conn(), models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository:  "acme/bb",
		VCSProvider: models.ProviderBitbucket,
	})
	require.NoError(t, err)

	otherFactory := createContextRepoWorkspace(t, other.ID, otherIntegration.ID)
	_, err = otherFactory.CreateAgentResource(database.Conn(), models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/other",
	})
	require.NoError(t, err)

	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		mocks.GitHubResponse(http.StatusCreated, `{"token":"ghs_write","expires_at":"2030-01-01T00:00:00Z"}`),
		mocks.GitHubResponse(http.StatusCreated, `{"token":"ghs_read","expires_at":"2030-01-01T00:00:00Z"}`),
	}}
	secrets, err := (&GitHub{}).ResolveSecrets(core.IntegrationSecretContext{
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{
			IntegrationID: integration.ID.String(),
			Metadata:      common.Metadata{HostedApp: true},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, runner.GitHubContextTokenEnv, integrationSecretGitHubContextToken)
	assert.Equal(t, []byte("ghs_write"), secrets.Values[integrationSecretGitHubToken])
	assert.Equal(t, []byte("ghs_read"), secrets.Values[integrationSecretGitHubContextToken])
	assert.Contains(t, secrets.Usage, "read-only token")
	require.Len(t, httpCtx.Requests, 2)
	writeBody := readRequestBody(t, httpCtx.Requests[0])
	readBody := readRequestBody(t, httpCtx.Requests[1])
	assert.JSONEq(t, `{"repository_ids":[601]}`, writeBody)
	assert.JSONEq(t, `{"repository_ids":[602],"permissions":{"contents":"read","metadata":"read"}}`, readBody)
}

func TestGitHub__ResolveSecrets__HostedContextRepositoryRefresh(t *testing.T) {
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
	require.NoError(t, models.ReplaceVCSProviderRepositories(database.Conn(), models.ProviderGitHub, 501, []models.VCSProviderRepository{
		{RepositoryID: 601, FullName: "acme/api"},
		{RepositoryID: 602, FullName: "acme/web"},
	}))
	require.NoError(t, models.GrantVCSProviderBindingRepository(database.Conn(), integration.ID, models.ProviderGitHub, 601))

	factory := createContextRepoWorkspace(t, organization.ID, integration.ID)
	_, err = factory.CreateAgentResource(database.Conn(), models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/web",
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(database.Conn(), models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/missing",
	})
	require.NoError(t, err)

	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		mocks.GitHubResponse(http.StatusCreated, `{"token":"ghs_write","expires_at":"2030-01-01T00:00:00Z"}`),
		mocks.GitHubResponse(http.StatusCreated, `{"token":"ghs_read","expires_at":"2030-01-01T00:00:00Z"}`),
	}}
	secrets, err := (&GitHub{}).ResolveSecrets(core.IntegrationSecretContext{
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{
			IntegrationID: integration.ID.String(),
			Metadata:      common.Metadata{HostedApp: true},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, []byte("ghs_read"), secrets.Values[integrationSecretGitHubContextToken])
	require.Len(t, httpCtx.Requests, 2)
	assert.JSONEq(t, `{"repository_ids":[602],"permissions":{"contents":"read","metadata":"read"}}`, readRequestBody(t, httpCtx.Requests[1]))
	var jobs int64
	require.NoError(t, database.Conn().Model(&models.VCSProviderInstallationReconcileJob{}).
		Where("provider = ? AND installation_id = ?", models.ProviderGitHub, int64(501)).
		Count(&jobs).Error)
	assert.Equal(t, int64(1), jobs)
}

func createContextRepoWorkspace(t *testing.T, organizationID, integrationID uuid.UUID) *models.Factory {
	t.Helper()
	factory, err := models.CreateFactory(database.Conn(), organizationID, "workspace-"+uuid.NewString(), "", "")
	require.NoError(t, err)
	integration := integrationID.String()
	require.NoError(t, factory.UpdateOnboarding(database.Conn(), models.FactoryOnboardingPatch{
		VCSIntegrationID: &integration,
	}))
	return factory
}

func readRequestBody(t *testing.T, request *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	return string(body)
}
