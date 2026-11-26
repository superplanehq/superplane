package contexts

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestIntegrationContextHostedBindingUsesTransaction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	t.Setenv(githubcommon.EnvGitHubAppID, "99")
	t.Setenv(githubcommon.EnvGitHubAppSlug, "superplane")
	t.Setenv(githubcommon.EnvGitHubAppPrivateKey, "test-pem")
	t.Setenv(githubcommon.EnvGitHubAppWebhookSecret, "test-secret")
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider: models.ProviderGitHub, InstallationID: 501, AccountLogin: "acme",
	}))
	integration, err := models.FindOrCreateVCSProviderBinding(db, r.Organization.ID, models.ProviderGitHub, 501, "acme")
	require.NoError(t, err)
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, 501, []models.VCSProviderRepository{
		{RepositoryID: 601, FullName: "acme/previous"},
		{RepositoryID: 602, FullName: "acme/primary"},
		{RepositoryID: 603, FullName: "acme/context"},
	}))
	require.NoError(t, models.GrantVCSProviderBindingRepository(db, integration.ID, models.ProviderGitHub, 601))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	integrationID, primary := integration.ID.String(), "acme/primary"
	primaryID := int64(602)
	rollback := errors.New("roll back repository settings")
	err = db.Transaction(func(tx *gorm.DB) error {
		require.NoError(t, factory.UpdateOnboarding(tx, models.FactoryOnboardingPatch{
			VCSIntegrationID: &integrationID, AppRepository: &primary, AppRepositoryID: &primaryID,
		}))
		require.NoError(t, models.SyncVCSProviderBindingRepositories(tx, integration.ID, models.ProviderGitHub))
		for _, repository := range []string{"acme/context", "acme/missing"} {
			_, err := factory.CreateAgentResource(tx, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
				Repository: repository, Access: models.FactoryContextRepoAccessWrite,
			})
			require.NoError(t, err)
		}
		binding, err := githubcommon.ResolveHostedAppBinding(NewIntegrationContext(tx, nil, integration, r.Encryptor, r.Registry, nil))
		require.NoError(t, err)
		require.ElementsMatch(t, []int64{602, 603}, binding.RepositoryIDs)
		var jobs int64
		require.NoError(t, tx.Model(&models.VCSProviderInstallationReconcileJob{}).Where("installation_id = ?", 501).Count(&jobs).Error)
		require.EqualValues(t, 1, jobs)
		return rollback
	})
	require.ErrorIs(t, err, rollback)
	binding, err := githubcommon.ResolveHostedAppBinding(NewIntegrationContext(db, nil, integration, r.Encryptor, r.Registry, nil))
	require.NoError(t, err)
	require.Equal(t, []int64{601}, binding.RepositoryIDs)
	var jobs int64
	require.NoError(t, db.Model(&models.VCSProviderInstallationReconcileJob{}).Where("installation_id = ?", 501).Count(&jobs).Error)
	require.Zero(t, jobs, "a failed configuration must not schedule a catalog refresh")
}
