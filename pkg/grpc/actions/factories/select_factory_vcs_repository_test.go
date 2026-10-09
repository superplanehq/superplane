package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestSelectFactoryVCSProviderRepositoryWithoutAccountConnection(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-test")
	t.Setenv(config.EnvGitHubAppPrivateKey, "test-private-key")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "test-webhook-secret")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")

	r := support.Setup(t)
	db := database.DB(t.Context())
	const installationID = int64(303)
	const repositoryID = int64(404)
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: installationID,
		AccountLogin:   "acme",
		AccountType:    "Organization",
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, installationID, []models.VCSProviderRepository{{
		RepositoryID:  repositoryID,
		FullName:      "acme/api",
		DefaultBranch: "main",
	}}))

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	response, err := SelectFactoryVCSProviderRepository(ctx, r.Organization.ID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{
		Id:           factory.ID.String(),
		Provider:     models.ProviderGitHub,
		RepositoryId: repositoryID,
	})
	require.NoError(t, err)
	assert.Equal(t, "acme/api", response.Factory.Onboarding.AppRepository)
	assert.Equal(t, repositoryID, response.Factory.Onboarding.AppRepositoryId)
}

func TestSelectFactoryVCSProviderRepository(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-test")
	t.Setenv(config.EnvGitHubAppPrivateKey, "test-private-key")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "test-webhook-secret")
	t.Setenv("GITHUB_CLIENT_ID", "Iv1.env")
	t.Setenv("GITHUB_CLIENT_SECRET", "env-secret")

	r := support.Setup(t)
	db := database.DB(t.Context())
	const githubUserID = int64(42)
	const installationID = int64(101)
	const repositoryID = int64(201)
	const secondRepositoryID = int64(202)

	require.NoError(t, models.SaveAccountLinkedAccount(db, models.NewAccountLinkedAccount(
		r.Account.ID,
		models.ProviderGitHub,
		"42",
		"octocat",
		"The Octocat",
		"https://github.com/octocat.png",
	)))
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: installationID,
		AccountLogin:   "acme",
		AccountType:    "Organization",
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, installationID, []models.VCSProviderRepository{
		{
			RepositoryID:  repositoryID,
			FullName:      "acme/api",
			Private:       true,
			DefaultBranch: "main",
		},
		{
			RepositoryID:  secondRepositoryID,
			FullName:      "acme/web",
			Private:       true,
			DefaultBranch: "trunk",
		},
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositoryCollaborators(db, models.ProviderGitHub, repositoryID, []models.VCSProviderRepositoryCollaborator{{
		ProviderUserID: githubUserID,
		ProviderLogin:  "octocat",
	}}))
	require.NoError(t, models.ReplaceVCSProviderRepositoryCollaborators(db, models.ProviderGitHub, secondRepositoryID, []models.VCSProviderRepositoryCollaborator{{
		ProviderUserID: githubUserID,
		ProviderLogin:  "octocat",
	}}))

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("saves the global repository and reuses the organization binding", func(t *testing.T) {
		first, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		second, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		firstResponse, err := SelectFactoryVCSProviderRepository(ctx, r.Organization.ID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{
			Id:           first.ID.String(),
			Provider:     models.ProviderGitHub,
			RepositoryId: repositoryID,
		})
		require.NoError(t, err)
		secondResponse, err := SelectFactoryVCSProviderRepository(ctx, r.Organization.ID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{
			Id:           second.ID.String(),
			Provider:     models.ProviderGitHub,
			RepositoryId: repositoryID,
		})
		require.NoError(t, err)

		firstOnboarding := firstResponse.Factory.Onboarding
		secondOnboarding := secondResponse.Factory.Onboarding
		assert.Equal(t, repositoryID, firstOnboarding.AppRepositoryId)
		assert.Equal(t, repositoryID, firstOnboarding.BacklogRepositoryId)
		assert.Equal(t, "acme/api", firstOnboarding.AppRepository)
		assert.Equal(t, "acme/api", firstOnboarding.BacklogRepository)
		assert.Equal(t, "main", firstOnboarding.DefaultBranch)
		assert.Equal(t, firstOnboarding.VcsIntegrationId, secondOnboarding.VcsIntegrationId)

		bindings, err := models.ListVCSProviderIntegrationBindings(db, models.ProviderGitHub, installationID)
		require.NoError(t, err)
		require.Len(t, bindings, 1)
		assert.Equal(t, firstOnboarding.VcsIntegrationId, bindings[0].IntegrationID.String())

		granted, err := models.ListVCSProviderBindingRepositories(db, bindings[0].IntegrationID)
		require.NoError(t, err)
		require.Len(t, granted, 1)
		assert.Equal(t, repositoryID, granted[0].RepositoryID)

		_, err = SelectFactoryVCSProviderRepository(ctx, r.Organization.ID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{
			Id:           first.ID.String(),
			Provider:     models.ProviderGitHub,
			RepositoryId: secondRepositoryID,
		})
		require.NoError(t, err)
		granted, err = models.ListVCSProviderBindingRepositories(db, bindings[0].IntegrationID)
		require.NoError(t, err)
		require.Len(t, granted, 2)

		_, err = SelectFactoryVCSProviderRepository(ctx, r.Organization.ID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{
			Id:           second.ID.String(),
			Provider:     models.ProviderGitHub,
			RepositoryId: secondRepositoryID,
		})
		require.NoError(t, err)
		granted, err = models.ListVCSProviderBindingRepositories(db, bindings[0].IntegrationID)
		require.NoError(t, err)
		require.Len(t, granted, 1)
		assert.Equal(t, secondRepositoryID, granted[0].RepositoryID)
	})

	t.Run("rejects a repository without cached push access", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, err = SelectFactoryVCSProviderRepository(ctx, r.Organization.ID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{
			Id:           factory.ID.String(),
			Provider:     models.ProviderGitHub,
			RepositoryId: 999,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
	})
}
