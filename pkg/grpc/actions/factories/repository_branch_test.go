package factories

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestChooseVerifiedDefaultBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		requested       string
		saved           string
		exists          bool
		verifyErr       error
		live            string
		liveErr         error
		repositoryEmpty bool
		want            string
		keepSaved       bool
		wantErr         string
	}{
		{name: "empty request keeps the saved branch", requested: " ", saved: "develop", keepSaved: true},
		{name: "existing branch is saved", requested: "master", exists: true, want: "master"},
		{name: "missing branch uses the provider default", requested: "main", live: " master ", want: "master"},
		{name: "failed lookup rejects a different requested branch", requested: "main", saved: "develop", verifyErr: errors.New("down"), wantErr: "could not verify the default branch"},
		{name: "failed lookup keeps the same saved branch", requested: "develop", saved: " develop ", verifyErr: errors.New("down"), keepSaved: true},
		{name: "failed lookup with nothing saved is rejected", requested: "main", verifyErr: errors.New("down"), wantErr: "could not verify the default branch"},
		{name: "failed provider default rejects a different requested branch", requested: "main", saved: "develop", liveErr: errors.New("down"), wantErr: "could not verify the default branch"},
		{name: "failed provider default keeps the same saved branch", requested: "develop", saved: "develop", liveErr: errors.New("down"), keepSaved: true},
		{name: "failed provider default with nothing saved is rejected", requested: "main", liveErr: errors.New("down"), wantErr: "could not verify the default branch"},
		{name: "missing provider default keeps a saved branch", requested: "main", saved: "develop", keepSaved: true},
		{name: "missing provider default with nothing saved is rejected", requested: "main", wantErr: "default branch is required"},
		{name: "empty repository accepts the requested branch", requested: "develop", repositoryEmpty: true, want: "develop"},
		{name: "empty repository keeps the requested branch when a default is reported", requested: "develop", live: "main", repositoryEmpty: true, want: "develop"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, keepSaved, err := chooseVerifiedDefaultBranch(test.requested, test.saved, onboardingBranchCheck{
				exists:          test.exists,
				live:            test.live,
				repositoryEmpty: test.repositoryEmpty,
				verifyErr:       test.verifyErr,
				liveErr:         test.liveErr,
			})
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.keepSaved, keepSaved)
		})
	}
}

func TestKeepSavedDefaultBranch(t *testing.T) {
	t.Parallel()

	saved := models.FactoryOnboardingConfig{AppRepository: "acme/api", DefaultBranch: "develop"}
	assert.Equal(t, "master", keepSavedDefaultBranch(saved, "acme/api", "master"))
	assert.Equal(t, "develop", keepSavedDefaultBranch(saved, "acme/api", " "))
	assert.Equal(t, "", keepSavedDefaultBranch(saved, "acme/web", ""))
}

func TestDefaultBranchFromCacheOrProvider(t *testing.T) {
	t.Parallel()

	called := false
	assert.Equal(t, "master", defaultBranchFromCacheOrProvider(" trunk ", func() (string, error) {
		called = true
		return "master", nil
	}))
	assert.True(t, called)
	assert.Equal(t, "trunk", defaultBranchFromCacheOrProvider(" trunk ", func() (string, error) {
		return "", errors.New("unavailable")
	}))
	assert.Equal(t, "trunk", defaultBranchFromCacheOrProvider(" trunk ", func() (string, error) {
		return " ", nil
	}))
	assert.Equal(t, "master", defaultBranchFromCacheOrProvider(" ", func() (string, error) {
		return " master ", nil
	}))
	assert.Equal(t, "", defaultBranchFromCacheOrProvider("", func() (string, error) {
		return "", errors.New("unavailable")
	}))
	assert.Equal(t, "trunk", defaultBranchFromCacheOrProvider("trunk", nil))
	assert.Equal(t, "", defaultBranchFromCacheOrProvider("", nil))
}

func TestGitHubInstallationForRepositorySkipsCatalogWithoutBinding(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 101,
		AccountLogin:   "acme",
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, 101, []models.VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/app", DefaultBranch: "main"},
	}))
	integrationID := createReadyGitHubEnterpriseIntegration(t, r.Organization.ID, "acme.ghe.com")
	integration, err := models.FindIntegrationInTransaction(db, r.Organization.ID, uuid.MustParse(integrationID))
	require.NoError(t, err)
	assert.Equal(t, models.ProviderGitHub, integration.AppName)

	installationID, repositoryID := githubInstallationForRepository(
		t.Context(),
		db,
		r.Organization.ID.String(),
		integration,
		"acme/app",
	)
	assert.Equal(t, int64(0), installationID)
	assert.Equal(t, int64(0), repositoryID)
}
