package pulls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

const pullRequestResponse = `{"id": 42, "number": 7, "html_url": "https://github.com/acme/api/pull/7"}`

type openAsUserFixture struct {
	organization *models.Organization
	integration  *contexts.IntegrationContext
	encryptor    crypto.Encryptor
}

func setupOpenAsUser(t *testing.T) *openAsUserFixture {
	t.Helper()
	require.NoError(t, database.TruncateTables())

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	t.Setenv(config.EnvGitHubAppID, "99")
	t.Setenv(config.EnvGitHubAppSlug, "superplane")
	t.Setenv(config.EnvGitHubAppPrivateKey, string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})))
	t.Setenv(config.EnvGitHubAppWebhookSecret, "whsec")
	t.Setenv(config.EnvGitHubAppClientID, "Iv1.client")
	t.Setenv(config.EnvGitHubAppClientSecret, "client-secret")

	encryptor := crypto.NewNoOpEncryptor()
	previous, previousErr := crypto.ProcessEncryptor()
	crypto.SetProcessEncryptor(encryptor)
	t.Cleanup(func() {
		if previousErr != nil {
			previous = nil
		}
		crypto.SetProcessEncryptor(previous)
	})

	organization, err := models.CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)
	require.NoError(t, models.UpsertVCSProviderInstallation(database.Conn(), &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 501,
		AccountLogin:   "acme",
	}))
	binding, err := models.FindOrCreateVCSProviderBinding(database.Conn(), organization.ID, models.ProviderGitHub, 501, "acme")
	require.NoError(t, err)
	require.NoError(t, models.ReplaceVCSProviderRepositories(database.Conn(), models.ProviderGitHub, 501, []models.VCSProviderRepository{
		{RepositoryID: 601, FullName: "acme/api"},
	}))
	require.NoError(t, models.GrantVCSProviderBindingRepository(database.Conn(), binding.ID, models.ProviderGitHub, 601))

	return &openAsUserFixture{
		organization: organization,
		integration: &contexts.IntegrationContext{
			IntegrationID: binding.ID.String(),
			Metadata:      common.Metadata{HostedApp: true},
		},
		encryptor: encryptor,
	}
}

func (f *openAsUserFixture) member(t *testing.T, login string, authorized bool) *models.User {
	t.Helper()
	account, err := models.CreateAccount(login, login+"-"+uuid.NewString()+"@example.com")
	require.NoError(t, err)
	user, err := models.CreateUser(f.organization.ID, account.ID, account.Email, login)
	require.NoError(t, err)
	if !authorized {
		return user
	}

	expiresAt := time.Now().Add(time.Hour)
	require.NoError(t, githubapp.SaveUserAuthorization(
		t.Context(),
		database.DB(t.Context()),
		f.encryptor,
		account.ID,
		99,
		&githubapp.GitHubUser{ID: time.Now().UnixNano(), Login: login},
		&githubapp.UserTokens{AccessToken: "ghu_" + login, AccessTokenExpiresAt: &expiresAt},
	))
	return user
}

func (f *openAsUserFixture) execute(t *testing.T, httpCtx *contexts.HTTPContext, repository, openAsUsers string) (*contexts.MetadataContext, *contexts.ExecutionStateContext) {
	t.Helper()
	metadata := &contexts.MetadataContext{}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	require.NoError(t, (&CreatePullRequest{}).Execute(core.ExecutionContext{
		OrganizationID: f.organization.ID.String(),
		Integration:    f.integration,
		HTTP:           httpCtx,
		Metadata:       metadata,
		ExecutionState: state,
		Configuration: map[string]any{
			"repository":  repository,
			"head":        "feature",
			"base":        "main",
			"title":       "Add feature",
			"openAsUsers": openAsUsers,
		},
	}))
	return metadata, state
}

func TestCreatePullRequestOpensAsFirstAuthorizedAssignee(t *testing.T) {
	fixture := setupOpenAsUser(t)
	unauthorized := fixture.member(t, "ada", false)
	authorized := fixture.member(t, "grace", true)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		mocks.GitHubResponse(http.StatusCreated, pullRequestResponse),
	}}

	metadata, state := fixture.execute(t, httpCtx, "api", unauthorized.ID.String()+","+authorized.ID.String())

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "/repos/acme/api/pulls", httpCtx.Requests[0].URL.Path)
	assert.Equal(t, "Bearer ghu_grace", httpCtx.Requests[0].Header.Get("Authorization"))
	assert.Equal(t, map[string]any{"openedAs": OpenedAsUser, "openedAsLogin": "grace"}, metadata.Metadata)
	assert.Equal(t, "github.pullRequest", state.Type)
}

func TestCreatePullRequestFallsBackToAppWhenUserTokenIsRejected(t *testing.T) {
	fixture := setupOpenAsUser(t)
	assignee := fixture.member(t, "ada", true)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		mocks.GitHubResponse(http.StatusForbidden, `{"message":"Resource not accessible by integration"}`),
		mocks.GitHubResponse(http.StatusCreated, `{"token":"ghs_installation","expires_at":"2030-01-01T00:00:00Z"}`),
		mocks.GitHubResponse(http.StatusCreated, pullRequestResponse),
	}}

	metadata, state := fixture.execute(t, httpCtx, "api", assignee.ID.String())

	require.Len(t, httpCtx.Requests, 3)
	assert.Equal(t, "Bearer ghu_ada", httpCtx.Requests[0].Header.Get("Authorization"))
	assert.Equal(t, "/app/installations/501/access_tokens", httpCtx.Requests[1].URL.Path)
	assert.Equal(t, "/repos/acme/api/pulls", httpCtx.Requests[2].URL.Path)
	assert.Equal(t, map[string]any{"openedAs": OpenedAsBot}, metadata.Metadata)
	assert.Equal(t, "github.pullRequest", state.Type)
}

func TestCreatePullRequestIgnoresAssigneesForNonHostedIntegrations(t *testing.T) {
	fixture := setupOpenAsUser(t)
	assignee := fixture.member(t, "ada", true)
	fixture.integration = mocks.IntegrationContextForNewSetupFlow()
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		mocks.GitHubResponse(http.StatusCreated, pullRequestResponse),
	}}

	metadata, _ := fixture.execute(t, httpCtx, "api", assignee.ID.String())

	require.Len(t, httpCtx.Requests, 1)
	assert.NotEqual(t, "Bearer ghu_ada", httpCtx.Requests[0].Header.Get("Authorization"))
	assert.Equal(t, map[string]any{"openedAs": OpenedAsBot}, metadata.Metadata)
}

func appOpensPullRequest() *contexts.HTTPContext {
	return &contexts.HTTPContext{Responses: []*http.Response{
		mocks.GitHubResponse(http.StatusCreated, `{"token":"ghs_installation","expires_at":"2030-01-01T00:00:00Z"}`),
		mocks.GitHubResponse(http.StatusCreated, pullRequestResponse),
	}}
}

func assertOpenedByApp(t *testing.T, httpCtx *contexts.HTTPContext, metadata *contexts.MetadataContext) {
	t.Helper()
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "/app/installations/501/access_tokens", httpCtx.Requests[0].URL.Path)
	assert.Equal(t, map[string]any{"openedAs": OpenedAsBot}, metadata.Metadata)
}

func TestCreatePullRequestDoesNotUseMemberTokenOutsideGrantedRepositories(t *testing.T) {
	fixture := setupOpenAsUser(t)
	assignee := fixture.member(t, "ada", true)
	httpCtx := appOpensPullRequest()

	metadata, _ := fixture.execute(t, httpCtx, "acme/private-elsewhere", assignee.ID.String())

	assertOpenedByApp(t, httpCtx, metadata)
}

func TestCreatePullRequestSkipsBlockedAssignees(t *testing.T) {
	fixture := setupOpenAsUser(t)
	assignee := fixture.member(t, "ada", true)
	account, err := models.FindAccountByID(assignee.AccountID.String())
	require.NoError(t, err)
	require.NoError(t, account.Block(database.Conn(), time.Now()))
	httpCtx := appOpensPullRequest()

	metadata, _ := fixture.execute(t, httpCtx, "api", assignee.ID.String())

	assertOpenedByApp(t, httpCtx, metadata)
}

func TestCreatePullRequestInvalidatesRejectedMemberToken(t *testing.T) {
	fixture := setupOpenAsUser(t)
	assignee := fixture.member(t, "ada", true)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		mocks.GitHubResponse(http.StatusUnauthorized, `{"message":"Bad credentials"}`),
		mocks.GitHubResponse(http.StatusCreated, `{"token":"ghs_installation","expires_at":"2030-01-01T00:00:00Z"}`),
		mocks.GitHubResponse(http.StatusCreated, pullRequestResponse),
	}}

	metadata, _ := fixture.execute(t, httpCtx, "api", assignee.ID.String())

	assert.Equal(t, map[string]any{"openedAs": OpenedAsBot}, metadata.Metadata)
	authorization, err := models.FindAccountGitHubAuthorization(database.Conn(), *assignee.AccountID)
	require.NoError(t, err)
	assert.NotNil(t, authorization.InvalidatedAt)
}
