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
	require.NoError(t, worker.storeMerges(target, sync, from, time.Time{}, merged, completeVelocityRead(), now, now))
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
	require.NoError(t, worker.storeMerges(target, sync, from, time.Time{}, nil, completeVelocityRead(), now, now))
	rows, err = models.ListFactoryVelocityRepositoryMerges(db, factory.ID, from, now)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, models.ProviderGitHub, rows[0].Provider)
}

func TestFactoryVelocitySync_ShortBitbucketReadDoesNotFinish(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	from := now.Add(-60 * 24 * time.Hour)
	sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, now)
	require.NoError(t, err)
	target := models.FactoryVelocitySyncTarget{
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		VCSProvider:    models.ProviderBitbucket,
		Repository:     "acme/app",
	}
	cursor := now.Add(-2 * 24 * time.Hour)
	require.NoError(t, sync.RecordSuccess(db, target.Repository, now.Add(-time.Hour), from))
	merged := []repositoryMerge{{
		repository: target.Repository,
		number:     7,
		source:     models.FactoryVelocityMergeSourcePeople,
		authorUUID: uuid.NewString(),
		mergedAt:   now.Add(-time.Hour),
	}}
	worker := NewFactoryVelocitySyncWorker("", nil, nil)

	require.NoError(t, worker.storeMerges(
		target,
		sync,
		from,
		time.Time{},
		merged,
		velocityReadCoverage{ContinueBefore: cursor},
		now,
		now,
	))

	stored, err := models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.SyncedAt, "the first slice stamps when the newest pages were read")
	assert.WithinDuration(t, now, *stored.SyncedAt, time.Second)
	require.NotNil(t, stored.BackfilledFrom)
	assert.WithinDuration(t, cursor, *stored.BackfilledFrom, time.Second)
	assert.Equal(t, models.FactoryVelocitySyncStateBackfilling, stored.SyncState)
	assert.Equal(t, target.Repository, stored.Repository)

	later := now.Add(time.Minute)
	require.NoError(t, worker.storeMerges(
		target,
		stored,
		from,
		cursor,
		nil,
		velocityReadCoverage{ContinueBefore: cursor.Add(-24 * time.Hour)},
		later,
		later,
	))
	stored, err = models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.SyncedAt)
	assert.WithinDuration(t, now, *stored.SyncedAt, time.Second, "a later slice must not move the newest-page stamp forward")
}

func TestFactoryVelocitySync_FinishedOlderSliceWaitsForTheNewerWindow(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	from := now.Add(-60 * 24 * time.Hour)
	sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, now)
	require.NoError(t, err)
	target := models.FactoryVelocitySyncTarget{
		OrganizationID:   r.Organization.ID,
		FactoryID:        factory.ID,
		VCSProvider:      models.ProviderBitbucket,
		Repository:       "acme/app",
		SyncedRepository: "acme/app",
	}
	cursor := now.Add(-2 * 24 * time.Hour)
	worker := NewFactoryVelocitySyncWorker("", nil, nil)
	require.NoError(t, worker.storeMerges(
		target,
		sync,
		from,
		time.Time{},
		nil,
		velocityReadCoverage{ContinueBefore: cursor},
		now.Add(-time.Hour),
		now.Add(-time.Hour),
	))

	rebuilding := target
	rebuilding.SyncState = models.FactoryVelocitySyncStateBackfilling
	rebuilding.BackfilledFrom = &cursor
	require.NoError(t, worker.storeMerges(
		rebuilding,
		sync,
		from,
		cursor,
		nil,
		completeVelocityRead(),
		now,
		now,
	))

	stored, err := models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryVelocitySyncStateCatchUp, stored.SyncState)
	require.NotNil(t, stored.SyncedAt)
	assert.WithinDuration(t, now.Add(-time.Hour), *stored.SyncedAt, time.Second,
		"finishing the older slice must not stamp the sync as fresh")
	assert.True(t, stored.SyncInProgress())

	caught := target
	caught.SyncState = models.FactoryVelocitySyncStateCatchUp
	caught.SyncedAt = stored.SyncedAt
	caught.BackfilledFrom = stored.BackfilledFrom
	recent := repositoryMerge{
		repository: target.Repository,
		number:     9,
		source:     models.FactoryVelocityMergeSourcePeople,
		authorUUID: uuid.NewString(),
		mergedAt:   now.Add(-time.Minute),
	}
	require.NoError(t, worker.storeMerges(
		caught,
		stored,
		bitbucketCatchUpFrom(caught, now),
		time.Time{},
		[]repositoryMerge{recent},
		completeVelocityRead(),
		now,
		now,
	))

	stored, err = models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryVelocitySyncStateIdle, stored.SyncState)
	assert.False(t, stored.SyncInProgress())
	require.NotNil(t, stored.SyncedAt)
	assert.WithinDuration(t, now, *stored.SyncedAt, time.Second)
	rows, err := models.ListFactoryVelocityRepositoryMerges(db, factory.ID, from, now.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(9), rows[0].Number)
}

func TestFactoryVelocitySync_ShortCatchUpContinuesWithoutRereadingOlderHistory(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	from := now.Add(-60 * 24 * time.Hour)
	sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, now)
	require.NoError(t, err)
	target := models.FactoryVelocitySyncTarget{
		OrganizationID:   r.Organization.ID,
		FactoryID:        factory.ID,
		VCSProvider:      models.ProviderBitbucket,
		Repository:       "acme/app",
		SyncedRepository: "acme/app",
	}
	firstSlice := now.Add(-10 * 24 * time.Hour)
	worker := NewFactoryVelocitySyncWorker("", nil, nil)
	require.NoError(t, worker.storeMerges(
		target,
		sync,
		from,
		time.Time{},
		nil,
		velocityReadCoverage{ContinueBefore: firstSlice},
		firstSlice,
		firstSlice,
	))

	older := repositoryMerge{
		repository: target.Repository,
		number:     3,
		source:     models.FactoryVelocityMergeSourcePeople,
		authorUUID: uuid.NewString(),
		mergedAt:   now.Add(-20 * 24 * time.Hour),
	}
	rebuilding := target
	rebuilding.SyncState = models.FactoryVelocitySyncStateBackfilling
	rebuilding.BackfilledFrom = &firstSlice
	require.NoError(t, worker.storeMerges(
		rebuilding,
		sync,
		from,
		firstSlice,
		[]repositoryMerge{older},
		completeVelocityRead(),
		now.Add(-time.Hour),
		now.Add(-time.Hour),
	))

	caught := target
	caught.SyncState = models.FactoryVelocitySyncStateCatchUp
	caught.SyncedAt = sync.SyncedAt
	caught.BackfilledFrom = sync.BackfilledFrom
	cursor := now.Add(-24 * time.Hour)
	recent := repositoryMerge{
		repository: target.Repository,
		number:     9,
		source:     models.FactoryVelocityMergeSourcePeople,
		authorUUID: uuid.NewString(),
		mergedAt:   now.Add(-time.Hour),
	}
	require.NoError(t, worker.storeMerges(
		caught,
		sync,
		bitbucketCatchUpFrom(caught, now),
		time.Time{},
		[]repositoryMerge{recent},
		velocityReadCoverage{ContinueBefore: cursor},
		now,
		now,
	))

	stored, err := models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryVelocitySyncStateCatchingUp, stored.SyncState)
	require.NotNil(t, stored.BackfilledFrom)
	assert.WithinDuration(t, cursor, *stored.BackfilledFrom, time.Second)
	require.NotNil(t, stored.SyncedAt)
	assert.WithinDuration(t, firstSlice, *stored.SyncedAt, time.Second,
		"a short catch-up must keep the first-slice time so the next read does not shrink the window")
	assert.True(t, stored.SyncInProgress())

	rows, err := models.ListFactoryVelocityRepositoryMerges(db, factory.ID, from, now.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, rows, 2, "a short catch-up must not delete the older history it did not read")
}

func TestFactoryVelocitySync_ShortRecomputeKeepsTheOriginalWindowStart(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	history := now.AddDate(0, 0, -velocitySyncBackfillDays-1)
	sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, now)
	require.NoError(t, err)
	require.NoError(t, sync.RecordSuccess(db, "acme/app", now.Add(-time.Hour), history))
	sync.SyncState = models.FactoryVelocitySyncStateIdle
	target := models.FactoryVelocitySyncTarget{
		OrganizationID:   r.Organization.ID,
		FactoryID:        factory.ID,
		VCSProvider:      models.ProviderBitbucket,
		Repository:       "acme/app",
		SyncedRepository: "acme/app",
		SyncedAt:         sync.SyncedAt,
		BackfilledFrom:   sync.BackfilledFrom,
		SyncState:        models.FactoryVelocitySyncStateIdle,
	}
	windowStart := now.Add(-velocitySyncRecomputeWindow)
	cursor := now.Add(-5 * 24 * time.Hour)
	worker := NewFactoryVelocitySyncWorker("", nil, nil)
	require.NoError(t, worker.storeMerges(
		target,
		sync,
		windowStart,
		time.Time{},
		nil,
		velocityReadCoverage{ContinueBefore: cursor},
		now,
		now,
	))

	stored, err := models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.SyncedAt)
	assert.WithinDuration(t, windowStart, *stored.SyncedAt, time.Second,
		"a short seven-day read must save the window start, not the retry clock")
	require.NotNil(t, stored.BackfilledFrom)
	assert.WithinDuration(t, cursor, *stored.BackfilledFrom, time.Second)

	later := now.Add(24 * time.Hour)
	resumed := target
	resumed.SyncState = stored.SyncState
	resumed.SyncedAt = stored.SyncedAt
	resumed.BackfilledFrom = stored.BackfilledFrom
	from, until := velocitySyncBounds(resumed, later)
	assert.WithinDuration(t, windowStart, from, time.Second,
		"a retry must not skip the unread day at the start of the window")
	assert.WithinDuration(t, cursor, until, time.Second)

	require.NoError(t, worker.storeMerges(
		resumed,
		stored,
		later.Add(-velocitySyncRecomputeWindow),
		cursor,
		nil,
		velocityReadCoverage{ContinueBefore: cursor.Add(-24 * time.Hour)},
		later,
		later,
	))
	stored, err = models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.SyncedAt)
	assert.WithinDuration(t, windowStart, *stored.SyncedAt, time.Second,
		"a later slice must keep the original window start")
}
