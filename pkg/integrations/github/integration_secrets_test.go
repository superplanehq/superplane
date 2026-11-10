package github

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
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

func TestGitHub__ResolveSecrets__InstallationTokenPermissions(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		legacy      bool
		invalid     bool
		permissions string
		expected    string
	}{
		{name: "new app keeps runner rights without check or status publication", permissions: `{"checks":"write","contents":"write","pull_requests":"write","statuses":"write","actions":"write","metadata":"read"}`, expected: `{"checks":"read","contents":"write","pull_requests":"write","statuses":"read","actions":"write","metadata":"read"}`},
		{name: "legacy private app cannot publish checks", legacy: true, permissions: `{"checks":"write","contents":"write","issues":"write"}`, expected: `{"checks":"read","contents":"write","issues":"write"}`},
		{name: "existing read permission stays read", permissions: `{"checks":"read","contents":"write"}`, expected: `{"checks":"read","contents":"write"}`},
		{name: "app without checks keeps its existing permissions", permissions: `{"contents":"write","workflows":"write"}`, expected: `{"contents":"write","workflows":"write"}`},
		{name: "missing permissions cannot return an unrestricted token", permissions: `null`, invalid: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			integrationCtx := &contexts.IntegrationContext{
				NewSetupFlow: !scenario.legacy,
				CurrentProperties: map[string]any{
					common.PropertyAuthMethod:        common.AuthMethodApp,
					common.PropertyAppID:             "99",
					common.PropertyAppInstallationID: "501",
				},
				CurrentSecrets: map[string]core.IntegrationSecret{
					common.SecretAppPEM: {Name: common.SecretAppPEM, Value: githubPrivateKeyPEM(t)},
				},
				Metadata: common.Metadata{InstallationID: "501", GitHubApp: common.GitHubAppMetadata{ID: 99}},
			}
			if scenario.legacy {
				integrationCtx.CurrentSecrets[common.GitHubAppPEM] = core.IntegrationSecret{Name: common.GitHubAppPEM, Value: githubPrivateKeyPEM(t)}
			}
			httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{"permissions":`+scenario.permissions+`}`),
				mocks.GitHubResponse(http.StatusCreated, `{"token":"ghs_runner","expires_at":"`+time.Now().Add(time.Hour).UTC().Format(time.RFC3339)+`"}`),
			}}
			secrets, err := (&GitHub{}).ResolveSecrets(core.IntegrationSecretContext{HTTP: httpCtx, Integration: integrationCtx})
			if scenario.invalid {
				require.Error(t, err)
				assert.Empty(t, secrets.Values[integrationSecretGitHubToken])
				require.Len(t, httpCtx.Requests, 1)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []byte("ghs_runner"), secrets.Values[integrationSecretGitHubToken])
			require.Len(t, httpCtx.Requests, 2)
			assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
			assert.Equal(t, "/app/installations/501", httpCtx.Requests[0].URL.Path)
			assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
			assert.Equal(t, "/app/installations/501/access_tokens", httpCtx.Requests[1].URL.Path)
			assert.JSONEq(t, `{"permissions":`+scenario.expected+`}`, readRequestBody(t, httpCtx.Requests[1]))
		})
	}
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
		mocks.GitHubResponse(http.StatusOK, `{"permissions":{"checks":"write","contents":"write","pull_requests":"write"}}`),
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
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
	assert.Equal(t, "/app/installations/501/access_tokens", httpCtx.Requests[1].URL.Path)
	body, err := io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"repository_ids":[601],"permissions":{"checks":"read","contents":"write","pull_requests":"write"}}`, string(body))
	assert.NotContains(t, secrets.Values, integrationSecretGitHubContextToken)
}

func TestGitHub__ResolveSecrets__HostedTokensFollowContextRepositoryAccess(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	organization, err := models.CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)
	other, err := models.CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)
	require.NoError(t, models.EnableExperimentalFeature(organization.ID, features.FeatureFactoryMultiRepoTasks))
	require.NoError(t, models.EnableExperimentalFeature(other.ID, features.FeatureFactoryMultiRepoTasks))
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
		{RepositoryID: 606, FullName: "acme/docs"},
	}))

	factory := createContextRepoWorkspace(t, organization.ID, integration.ID)
	_, err = factory.CreateAgentResource(database.Conn(), models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/web",
		Access:     models.FactoryContextRepoAccessWrite,
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(database.Conn(), models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/docs",
		Access:     models.FactoryContextRepoAccessRead,
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
		Access:     models.FactoryContextRepoAccessWrite,
	})
	require.NoError(t, err)
	require.NoError(t, models.SyncVCSProviderBindingRepositories(database.Conn(), integration.ID, models.ProviderGitHub))

	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		mocks.GitHubResponse(http.StatusOK, `{"permissions":{"checks":"write","contents":"write"}}`),
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
	require.Len(t, httpCtx.Requests, 3)
	assert.JSONEq(t, `{"repository_ids":[601,602],"permissions":{"checks":"read","contents":"write"}}`, readRequestBody(t, httpCtx.Requests[1]))
	assert.JSONEq(t, `{"repository_ids":[606],"permissions":{"contents":"read","metadata":"read"}}`, readRequestBody(t, httpCtx.Requests[2]))
}

func TestGitHub__ResolveSecrets__HostedWriteTokenCoversContextRepositoriesWithoutGrants(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	organization, err := models.CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)
	require.NoError(t, models.EnableExperimentalFeature(organization.ID, features.FeatureFactoryMultiRepoTasks))
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
	// A connection saved before agents changed context repositories has a
	// grant for the primary repository only.
	require.NoError(t, models.GrantVCSProviderBindingRepository(database.Conn(), integration.ID, models.ProviderGitHub, 601))

	factory := createContextRepoWorkspace(t, organization.ID, integration.ID)
	for _, repository := range []string{"acme/web", "acme/missing"} {
		_, err = factory.CreateAgentResource(database.Conn(), models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
			Repository: repository,
			Access:     models.FactoryContextRepoAccessWrite,
		})
		require.NoError(t, err)
	}

	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		mocks.GitHubResponse(http.StatusOK, `{"permissions":{"checks":"write","contents":"write"}}`),
		mocks.GitHubResponse(http.StatusCreated, `{"token":"ghs_write","expires_at":"2030-01-01T00:00:00Z"}`),
	}}
	_, err = (&GitHub{}).ResolveSecrets(core.IntegrationSecretContext{
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{
			IntegrationID: integration.ID.String(),
			Metadata:      common.Metadata{HostedApp: true},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 2)
	assert.JSONEq(t, `{"repository_ids":[601,602],"permissions":{"checks":"read","contents":"write"}}`, readRequestBody(t, httpCtx.Requests[1]))
	var jobs int64
	require.NoError(t, database.Conn().Model(&models.VCSProviderInstallationReconcileJob{}).
		Where("provider = ? AND installation_id = ?", models.ProviderGitHub, int64(501)).
		Count(&jobs).Error)
	assert.Equal(t, int64(1), jobs, "a context repository missing from the catalog refreshes the installation")
}

func createContextRepoWorkspace(t *testing.T, organizationID, integrationID uuid.UUID) *models.Factory {
	t.Helper()
	factory, err := models.CreateFactory(database.Conn(), organizationID, "workspace-"+uuid.NewString(), "", "")
	require.NoError(t, err)
	integration := integrationID.String()
	repository := "acme/api"
	repositoryID := int64(601)
	require.NoError(t, factory.UpdateOnboarding(database.Conn(), models.FactoryOnboardingPatch{
		VCSIntegrationID: &integration,
		AppRepository:    &repository,
		AppRepositoryID:  &repositoryID,
	}))
	return factory
}

func readRequestBody(t *testing.T, request *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	return string(body)
}
