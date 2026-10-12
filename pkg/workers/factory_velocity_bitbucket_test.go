package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestFactoryVelocitySync_BitbucketFactory(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	provider := models.ProviderBitbucket
	integrationID := uuid.New().String()
	repository := "acme/app"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSProvider:      &provider,
		VCSIntegrationID: &integrationID,
		AppRepository:    &repository,
	}))

	worker := NewFactoryVelocitySyncWorker("", nil, nil)
	synced, err := worker.SyncFactory(t.Context(), factory.ID)
	require.NoError(t, err)
	assert.True(t, synced)

	// ponytail: the test worker has no registry, so the sync records the
	// connection failure instead of the old not-supported rejection
	stored, err := models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	assert.Contains(t, stored.Error, "unavailable")
}

func TestFactoryVelocitySync_BitbucketMergesKeepTheirProvider(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, now)
	require.NoError(t, err)
	target := models.FactoryVelocitySyncTarget{
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		VCSProvider:    models.ProviderBitbucket,
		Repository:     "acme/app",
	}
	githubMerge := models.NewFactoryVelocityRepositoryMerge(r.Organization.ID, factory.ID, target.Repository, 42, models.FactoryVelocityMergeSourcePeople, now.Add(-time.Hour))
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, factory.ID, models.ProviderGitHub, from, now, []models.FactoryVelocityRepositoryMerge{githubMerge}))
	worker := NewFactoryVelocitySyncWorker("", nil, nil)
	merged := []repositoryMerge{{repository: target.Repository, number: 42, source: models.FactoryVelocityMergeSourcePeople, authorUUID: uuid.NewString(), mergedAt: now.Add(-time.Hour)}}
	require.NoError(t, worker.storeMerges(target, sync, from, merged, now))
	rows, err := models.ListFactoryVelocityRepositoryMerges(db, factory.ID, from, now)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	var bitbucketMerge models.FactoryVelocityRepositoryMerge
	for _, row := range rows {
		if row.Provider == models.ProviderBitbucket {
			bitbucketMerge = row
		}
	}
	assert.Equal(t, merged[0].authorUUID, bitbucketMerge.AuthorUUID)
	require.NoError(t, worker.storeMerges(target, sync, from, nil, now))
	rows, err = models.ListFactoryVelocityRepositoryMerges(db, factory.ID, from, now)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, models.ProviderGitHub, rows[0].Provider)
}
