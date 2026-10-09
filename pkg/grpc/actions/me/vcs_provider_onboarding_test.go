package me

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/bitbucketapp"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestDescribeBitbucketOnboardingIncludesInstalledWorkspacesWithoutRepositories(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(config.EnvBitbucketForgeAppID, "ari:cloud:ecosystem::app/example")
	t.Setenv(config.EnvBitbucketForgeInstallURL, "https://developer.atlassian.com/console/install/example")
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())
	const accountID = "11111111-1111-1111-1111-111111111111"
	db := database.DB(ctx)
	require.NoError(t, models.SaveAccountLinkedAccount(db,
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderBitbucket, accountID, "ada-bb", "", "")))
	now := time.Now()
	_, err := models.SaveBitbucketForgeDelivery(db, models.BitbucketForgeDelivery{
		InstallationID: "installation-1", WorkspaceUUID: "22222222-2222-2222-2222-222222222222",
		WorkspaceSlug: "acme", InstallerAccountID: "another-admin", SystemToken: []byte("encrypted-token"),
		TokenExpiresAt: now.Add(time.Hour), DeliveredAt: now,
	})
	require.NoError(t, err)
	bitbucketapp.SetSystemTokenSource(func(string) (string, time.Time, error) { return "system-token", now.Add(time.Hour), nil })
	t.Cleanup(func() { bitbucketapp.SetSystemTokenSource(nil) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "Bearer system-token", request.Header.Get("Authorization"))
		assert.Equal(t, `user.uuid="{`+accountID+`}"`, request.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/workspaces/acme/permissions":
			_, _ = w.Write([]byte(`{"values":[{"permission":"member","workspace":{"slug":"acme"}}]}`))
		case "/workspaces/acme/permissions/repositories":
			_, _ = w.Write([]byte(`{"values":[]}`))
		default:
			t.Errorf("unexpected Bitbucket request: %s", request.URL.Path)
			http.NotFound(w, request)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(bitbucket.UseDirectory(bitbucket.Directory{BaseURL: server.URL, HTTP: server.Client()}))

	response, err := DescribeVCSProviderOnboarding(ctx, models.ProviderBitbucket)
	require.NoError(t, err)
	assert.Empty(t, response.Repositories)
	require.Len(t, response.InstalledWorkspaces, 1)
	assert.Equal(t, "acme", response.InstalledWorkspaces[0].Slug)
	assert.Equal(t, "installation-1", response.InstalledWorkspaces[0].InstallationId)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", response.InstalledWorkspaces[0].ExternalId)
}

func TestDescribeVCSProviderOnboardingListsLinkedIdentities(t *testing.T) {
	r := support.Setup(t)
	setGitHubAccountConnection(t)
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())

	require.NoError(t, models.SaveAccountLinkedAccount(
		database.Conn(),
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderGitHub, "101", "first-user", "", ""),
	))
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.Conn(),
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderGitHub, "202", "second-user", "", ""),
	))

	response, err := DescribeVCSProviderOnboarding(ctx, models.ProviderGitHub)
	require.NoError(t, err)
	require.NotNil(t, response.Identity)
	assert.Equal(t, int64(202), response.Identity.UserId)
	assert.Equal(t, "second-user", response.Identity.Login)
	require.Len(t, response.Identities, 2)
	assert.Equal(t, "second-user", response.Identities[0].Login)
	assert.Equal(t, "first-user", response.Identities[1].Login)
}

func TestSelectVCSProviderOnboardingIdentity(t *testing.T) {
	r := support.Setup(t)
	setGitHubAccountConnection(t)
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())
	db := database.Conn()

	require.NoError(t, models.SaveAccountLinkedAccount(
		db,
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderGitHub, "101", "first-user", "", ""),
	))
	require.NoError(t, models.SaveAccountLinkedAccount(
		db,
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderGitHub, "202", "second-user", "", ""),
	))
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 301,
		AccountLogin:   "acme",
		AccountType:    "Organization",
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, 301, []models.VCSProviderRepository{
		{RepositoryID: 401, FullName: "acme/first"},
		{RepositoryID: 402, FullName: "acme/second"},
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositoryCollaborators(db, models.ProviderGitHub, 401, []models.VCSProviderRepositoryCollaborator{
		{ProviderUserID: 101, ProviderLogin: "first-user"},
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositoryCollaborators(db, models.ProviderGitHub, 402, []models.VCSProviderRepositoryCollaborator{
		{ProviderUserID: 202, ProviderLogin: "second-user"},
	}))

	response, err := DescribeVCSProviderOnboarding(ctx, models.ProviderGitHub)
	require.NoError(t, err)
	require.Len(t, response.Repositories, 1)
	assert.Equal(t, "acme/second", response.Repositories[0].FullName)

	_, err = SelectVCSProviderOnboardingIdentity(ctx, models.ProviderGitHub, 101)
	require.NoError(t, err)

	response, err = DescribeVCSProviderOnboarding(ctx, models.ProviderGitHub)
	require.NoError(t, err)
	require.NotNil(t, response.Identity)
	assert.Equal(t, int64(101), response.Identity.UserId)
	assert.Equal(t, "first-user", response.Identity.Login)
	require.Len(t, response.Repositories, 1)
	assert.Equal(t, "acme/first", response.Repositories[0].FullName)
}

func TestSelectVCSProviderOnboardingIdentityRejectsUnlinkedIdentity(t *testing.T) {
	r := support.Setup(t)
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())

	_, err := SelectVCSProviderOnboardingIdentity(ctx, models.ProviderGitHub, 404)
	code, _, ok := grpcerrors.HandlerStatus(err)
	assert.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, code)
}

func TestStartVCSProviderInstallationSignsOrganizationState(t *testing.T) {
	r := support.Setup(t)
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())
	setVCSProviderGitHubAppEnvironment(t)
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.Conn(),
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderGitHub, "101", "octocat", "", ""),
	))

	response, err := StartVCSProviderInstallation(ctx, models.ProviderGitHub)
	require.NoError(t, err)
	installURL, err := url.Parse(response.Url)
	require.NoError(t, err)
	state := installURL.Query().Get("state")
	assert.NotContains(t, state, r.Organization.ID.String())
	organizationID, err := githubcommon.VerifyHostedAppInstallState("test-webhook-secret", state)
	require.NoError(t, err)
	assert.Equal(t, r.Organization.ID, organizationID)
}

func TestDescribeVCSProviderOnboardingAcceptsBitbucketAccountID(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(config.EnvBitbucketForgeAppID, "")
	t.Setenv(config.EnvBitbucketForgeInstallURL, "")
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())
	const accountID = "11111111-1111-1111-1111-111111111111"
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.Conn(),
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderBitbucket, "{"+accountID+"}", "ada-bb", "", ""),
	))

	response, err := DescribeVCSProviderOnboarding(ctx, models.ProviderBitbucket)
	require.NoError(t, err)
	require.NotNil(t, response.Identity)
	assert.Equal(t, int64(0), response.Identity.GetUserId())
	assert.Equal(t, accountID, response.Identity.GetProviderUserId())
	assert.Equal(t, "ada-bb", response.Identity.GetLogin())
	assert.False(t, response.GetProviderConfigured())
	assert.Empty(t, response.GetRepositories())
	assert.Empty(t, response.GetPendingRequests())
}

func TestStartBitbucketInstallationReturnsDistributionLink(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(config.EnvBitbucketForgeAppID, "ari:cloud:ecosystem::app/example")
	t.Setenv(config.EnvBitbucketForgeInstallURL, "https://developer.atlassian.com/console/install/example")
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.Conn(),
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderBitbucket, "11111111-1111-1111-1111-111111111111", "ada-bb", "", ""),
	))

	response, err := StartVCSProviderInstallation(ctx, models.ProviderBitbucket)
	require.NoError(t, err)
	assert.Equal(t, "https://developer.atlassian.com/console/install/example", response.GetUrl())
	assert.NotContains(t, response.GetUrl(), "state=")
}

func TestRefreshBitbucketOnboardingDoesNotQueueGitHub(t *testing.T) {
	r := support.Setup(t)
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())

	response, err := RefreshVCSProviderOnboarding(ctx, models.ProviderBitbucket, nil)
	require.NoError(t, err)
	require.NotNil(t, response)

	var count int64
	require.NoError(t, database.Conn().Table("vcs_provider_reconcile_jobs").Count(&count).Error)
	assert.Zero(t, count)
}

func TestDescribeGitHubOnboardingSkipsAccountConnection(t *testing.T) {
	r := support.Setup(t)
	setVCSProviderGitHubAppEnvironment(t)
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())
	require.NoError(t, models.UpsertVCSProviderInstallation(database.Conn(), &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 101,
		AccountLogin:   "acme",
		AccountType:    "Organization",
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(database.Conn(), models.ProviderGitHub, 101, []models.VCSProviderRepository{{
		RepositoryID:  201,
		FullName:      "acme/api",
		DefaultBranch: "main",
	}}))

	response, err := DescribeVCSProviderOnboarding(ctx, models.ProviderGitHub)
	require.NoError(t, err)
	assert.True(t, response.GetProviderConfigured())
	assert.False(t, response.GetAccountConnectionRequired())
	assert.Nil(t, response.Identity)
	require.Len(t, response.GetRepositories(), 1)
	assert.Equal(t, "acme/api", response.GetRepositories()[0].GetFullName())
	assert.Equal(t, "acme", response.GetRepositories()[0].GetAccountLogin())
}

func TestStartGitHubInstallationWithoutAccountConnection(t *testing.T) {
	r := support.Setup(t)
	setVCSProviderGitHubAppEnvironment(t)
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())

	response, err := StartVCSProviderInstallation(ctx, models.ProviderGitHub)
	require.NoError(t, err)
	installURL, err := url.Parse(response.GetUrl())
	require.NoError(t, err)
	assert.Equal(t, "github.com", installURL.Host)
	assert.NotEmpty(t, installURL.Query().Get("state"))
}

func setGitHubAccountConnection(t *testing.T) {
	t.Helper()
	t.Setenv("GITHUB_CLIENT_ID", "Iv1.env")
	t.Setenv("GITHUB_CLIENT_SECRET", "env-secret")
}

func setVCSProviderGitHubAppEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv(githubcommon.EnvGitHubAppID, "12345")
	t.Setenv(githubcommon.EnvGitHubAppSlug, "superplane")
	t.Setenv(
		githubcommon.EnvGitHubAppPrivateKey,
		"-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----",
	)
	t.Setenv(githubcommon.EnvGitHubAppWebhookSecret, "test-webhook-secret")
}
