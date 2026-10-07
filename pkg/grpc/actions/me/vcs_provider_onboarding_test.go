package me

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestDescribeVCSProviderOnboardingListsLinkedIdentities(t *testing.T) {
	r := support.Setup(t)
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
