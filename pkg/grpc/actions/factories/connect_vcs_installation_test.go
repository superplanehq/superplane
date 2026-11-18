package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
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

func TestConnectVCSProviderInstallation(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-test")
	t.Setenv(config.EnvGitHubAppPrivateKey, "test-private-key")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "test-webhook-secret")
	t.Setenv("GITHUB_CLIENT_ID", "Iv1.env")
	t.Setenv("GITHUB_CLIENT_SECRET", "env-secret")

	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.SaveAccountLinkedAccount(db, models.NewAccountLinkedAccount(
		r.Account.ID, models.ProviderGitHub, "42", "octocat", "The Octocat", "https://github.com/octocat.png",
	)))
	for _, installation := range []struct {
		id           int64
		login        string
		repositoryID int64
		collaborator bool
	}{
		{id: 101, login: "acme", repositoryID: 201, collaborator: true},
		{id: 102, login: "other", repositoryID: 202},
	} {
		require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
			Provider: models.ProviderGitHub, InstallationID: installation.id, AccountLogin: installation.login, AccountType: "Organization",
		}))
		require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, installation.id, []models.VCSProviderRepository{{
			RepositoryID: installation.repositoryID, FullName: installation.login + "/api", DefaultBranch: "main",
		}}))
		if installation.collaborator {
			require.NoError(t, models.ReplaceVCSProviderRepositoryCollaborators(db, models.ProviderGitHub, installation.repositoryID,
				[]models.VCSProviderRepositoryCollaborator{{ProviderUserID: 42, ProviderLogin: "octocat"}}))
		}
	}
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("connects the installation of an accessible repository", func(t *testing.T) {
		response, err := ConnectVCSProviderInstallation(ctx, r.Organization.ID.String(), &pb.ConnectVCSProviderInstallationRequest{RepositoryId: 201})
		require.NoError(t, err)

		integration, err := models.FindIntegrationInTransaction(db, r.Organization.ID, uuid.MustParse(response.IntegrationId))
		require.NoError(t, err)
		assert.Equal(t, models.ProviderGitHub, integration.AppName)
		binding, err := models.FindVCSProviderIntegrationBinding(db, integration.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(101), binding.InstallationID)
	})

	t.Run("rejects an installation the caller cannot access", func(t *testing.T) {
		_, err := ConnectVCSProviderInstallation(ctx, r.Organization.ID.String(), &pb.ConnectVCSProviderInstallationRequest{RepositoryId: 202})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
	})
}
