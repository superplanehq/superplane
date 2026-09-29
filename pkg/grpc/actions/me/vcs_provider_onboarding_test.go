package me

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
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
