package models_test

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

func Test__FactoryVelocityRepositoryMerge__ReplaceIsIdempotent(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	from := now.Add(-48 * time.Hour)
	to := now.Add(time.Hour)

	merge := models.NewFactoryVelocityRepositoryMerge(r.Organization.ID, factory.ID, "Example/Repo", 7, models.FactoryVelocityMergeSourcePeople, now.Add(-time.Hour))
	merge.AuthorLogin = "octocat"

	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, factory.ID, from, to, []models.FactoryVelocityRepositoryMerge{merge}))
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, factory.ID, from, to, []models.FactoryVelocityRepositoryMerge{merge}))

	stored, err := models.ListFactoryVelocityRepositoryMerges(db, factory.ID, from, to)
	require.NoError(t, err)
	require.Len(t, stored, 1, "re-syncing a window must not duplicate a merge")
	assert.Equal(t, "example/repo", stored[0].Repository, "the repository is stored lowercase so lookups match")
	assert.Equal(t, int64(7), stored[0].Number)
	assert.Equal(t, "octocat", stored[0].AuthorLogin)
}

func Test__FactoryVelocityRepositoryMerge__ReplaceOnlyTouchesItsWindow(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	old := models.NewFactoryVelocityRepositoryMerge(r.Organization.ID, factory.ID, "example/repo", 1, models.FactoryVelocityMergeSourcePeople, now.Add(-30*24*time.Hour))
	old.AuthorLogin = "octocat"
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(
		db, factory.ID, now.Add(-60*24*time.Hour), now, []models.FactoryVelocityRepositoryMerge{old},
	))

	// A later sync recomputes only the last two days, and must leave history alone.
	recent := models.NewFactoryVelocityRepositoryMerge(r.Organization.ID, factory.ID, "example/repo", 2, models.FactoryVelocityMergeSourcePeople, now.Add(-time.Hour))
	recent.AuthorLogin = "hubber"
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(
		db, factory.ID, now.Add(-48*time.Hour), now.Add(time.Hour), []models.FactoryVelocityRepositoryMerge{recent},
	))

	stored, err := models.ListFactoryVelocityRepositoryMerges(db, factory.ID, now.Add(-60*24*time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, stored, 2)
	assert.Equal(t, int64(1), stored[0].Number, "the merge outside the recomputed window survives")
	assert.Equal(t, int64(2), stored[1].Number)
}

func Test__FactoryVelocityRepositoryMerge__ReplaceDropsMergesThatBecameSuperPlanes(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	from := now.Add(-48 * time.Hour)
	to := now.Add(time.Hour)

	merge := models.NewFactoryVelocityRepositoryMerge(r.Organization.ID, factory.ID, "example/repo", 9, models.FactoryVelocityMergeSourcePeople, now.Add(-time.Hour))
	merge.AuthorLogin = "octocat"
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, factory.ID, from, to, []models.FactoryVelocityRepositoryMerge{merge}))

	// The next sync recognizes the pull request as SuperPlane's and omits it.
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, factory.ID, from, to, nil))

	stored, err := models.ListFactoryVelocityRepositoryMerges(db, factory.ID, from, to)
	require.NoError(t, err)
	assert.Empty(t, stored, "a merge reclassified as SuperPlane work stops counting as people output")
}

func Test__ListFactoryPullRequestNumbers__IgnoresStateAndRepositoryCase(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "order", "", nil, nil, nil)
	require.NoError(t, err)

	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:   "https://github.com/example/repo/pull/4",
		State: models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	mergedAt := time.Now().Add(-time.Hour)
	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:      "https://github.com/example/repo/pull/5",
		State:    models.FactoryPullRequestStateMerged,
		MergedAt: &mergedAt,
	})
	require.NoError(t, err)

	numbers, err := models.ListFactoryPullRequestNumbers(db, factory.ID, "Example/Repo")
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{4, 5}, numbers,
		"an open pull request is still SuperPlane's, so its merge must not count as people output")
}

func Test__ListFactoryVelocityRepositoryMerges__ScopesToFactory(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	first, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	from := now.Add(-48 * time.Hour)
	to := now.Add(time.Hour)

	mine := models.NewFactoryVelocityRepositoryMerge(r.Organization.ID, first.ID, "example/repo", 1, models.FactoryVelocityMergeSourcePeople, now.Add(-time.Hour))
	mine.AuthorLogin = "octocat"
	theirs := models.NewFactoryVelocityRepositoryMerge(r.Organization.ID, second.ID, "example/repo", 1, models.FactoryVelocityMergeSourcePeople, now.Add(-time.Hour))
	theirs.AuthorLogin = "hubber"

	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, first.ID, from, to, []models.FactoryVelocityRepositoryMerge{mine}))
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, second.ID, from, to, []models.FactoryVelocityRepositoryMerge{theirs}))

	stored, err := models.ListFactoryVelocityRepositoryMerges(db, first.ID, from, to)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, "octocat", stored[0].AuthorLogin)
}

func Test__DeleteFactoryVelocityRepositoryMerges(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	merge := models.NewFactoryVelocityRepositoryMerge(r.Organization.ID, factory.ID, "example/repo", 1, models.FactoryVelocityMergeSourcePeople, now.Add(-time.Hour))
	merge.AuthorLogin = "octocat"
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(
		db, factory.ID, now.Add(-48*time.Hour), now.Add(time.Hour), []models.FactoryVelocityRepositoryMerge{merge},
	))

	require.NoError(t, models.DeleteFactoryVelocityRepositoryMerges(db, factory.ID))

	stored, err := models.ListFactoryVelocityRepositoryMerges(db, factory.ID, now.Add(-48*time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	assert.Empty(t, stored)
}

func Test__FactoryVelocitySync__ClaimIsExclusiveUntilTheLeaseExpires(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	// The worker offers "claimable if untouched since now minus the lease".
	horizon := time.Now().Add(-15 * time.Minute)

	claimed, err := models.ClaimFactoryVelocitySync(db, factory.ID, horizon)
	require.NoError(t, err)
	require.NotNil(t, claimed, "the first worker takes the workspace")

	again, err := models.ClaimFactoryVelocitySync(db, factory.ID, horizon)
	require.NoError(t, err)
	assert.Nil(t, again, "a second worker must not sync the same workspace")

	expired, err := models.ClaimFactoryVelocitySync(db, factory.ID, time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, expired, "an expired lease releases the workspace")
}

func Test__FactoryVelocitySync__RecordErrorKeepsTheWatermark(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, time.Now())
	require.NoError(t, err)

	syncedAt := time.Now().Add(-time.Hour)
	backfilledFrom := syncedAt.Add(-60 * 24 * time.Hour)
	require.NoError(t, sync.RecordSuccess(db, "example/repo", syncedAt, backfilledFrom))
	require.NoError(t, sync.RecordError(db, "GitHub is unavailable"))

	stored, err := models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	assert.Equal(t, "GitHub is unavailable", stored.Error)
	require.NotNil(t, stored.SyncedAt)
	assert.WithinDuration(t, syncedAt, *stored.SyncedAt, time.Second,
		"a failure must not move the watermark, so the next tick retries the same window")
}

func Test__FactoryVelocitySync__RecordSuccessClearsTheError(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, time.Now())
	require.NoError(t, err)
	require.NoError(t, sync.RecordError(db, "GitHub is unavailable"))

	now := time.Now()
	require.NoError(t, sync.RecordSuccess(db, "example/repo", now, now.Add(-60*24*time.Hour)))

	stored, err := models.FindFactoryVelocitySync(db, factory.ID)
	require.NoError(t, err)
	assert.Empty(t, stored.Error)
	assert.Equal(t, "example/repo", stored.Repository)
	assert.True(t, stored.CoversRepository("Example/Repo"), "the repository match ignores case")
}

// allSyncTargets is a limit large enough that no workspace of this test hides
// behind the workspaces other tests created. Every workspace with a repository
// is a target, and an unsynced one sorts first, so a tight limit would test the
// page size instead of the filter.
const allSyncTargets = 10000

func Test__ListFactoryVelocitySyncTargets__NeedsIntegrationAndRepository(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	bare, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	ready, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	integrationID := uuid.New().String()
	repository := "example/repo"
	err = ready.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &integrationID,
		AppRepository:    &repository,
	})
	require.NoError(t, err)

	targets, err := models.ListFactoryVelocitySyncTargets(db, time.Now(), time.Now(), allSyncTargets)
	require.NoError(t, err)

	byFactory := make(map[uuid.UUID]models.FactoryVelocitySyncTarget, len(targets))
	for _, target := range targets {
		byFactory[target.FactoryID] = target
	}

	assert.NotContains(t, byFactory, bare.ID, "a workspace without a repository has nothing to sync")
	target, ok := byFactory[ready.ID]
	require.True(t, ok)
	assert.Equal(t, repository, target.Repository)
	assert.Equal(t, integrationID, target.IntegrationID.String())
	assert.Nil(t, target.SyncedAt)
	assert.Empty(t, target.SyncedRepository)
}

func Test__ListFactoryVelocitySyncTargets__SkipsFreshAndClaimedWorkspaces(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	integrationID := uuid.New().String()
	repository := "example/repo"
	err = factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &integrationID,
		AppRepository:    &repository,
	})
	require.NoError(t, err)

	sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, time.Now())
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, sync.RecordSuccess(db, repository, now, now.Add(-60*24*time.Hour)))

	fresh, err := models.ListFactoryVelocitySyncTargets(db, now.Add(-5*time.Minute), now.Add(-15*time.Minute), allSyncTargets)
	require.NoError(t, err)
	assert.NotContains(t, factoryIDs(fresh), factory.ID, "a workspace synced a moment ago waits its turn")

	due, err := models.ListFactoryVelocitySyncTargets(db, now.Add(time.Hour), now.Add(time.Hour), allSyncTargets)
	require.NoError(t, err)
	assert.Contains(t, factoryIDs(due), factory.ID)
}

func Test__FindFactoryVelocitySyncTarget(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	bare, err := models.FindFactoryVelocitySyncTarget(db, factory.ID)
	require.NoError(t, err)
	assert.Nil(t, bare, "a workspace without a repository has nothing to sync")

	integrationID := uuid.New().String()
	repository := "example/repo"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &integrationID,
		AppRepository:    &repository,
	}))

	// A workspace synced a moment ago is not due, but a user asking for a fresh
	// read must still find it.
	sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, time.Now())
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, sync.RecordSuccess(db, repository, now, now.Add(-60*24*time.Hour)))

	target, err := models.FindFactoryVelocitySyncTarget(db, factory.ID)
	require.NoError(t, err)
	require.NotNil(t, target)
	assert.Equal(t, repository, target.Repository)
	assert.Equal(t, integrationID, target.IntegrationID.String())
	assert.Equal(t, repository, target.SyncedRepository)

	missing, err := models.FindFactoryVelocitySyncTarget(db, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func factoryIDs(targets []models.FactoryVelocitySyncTarget) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, target.FactoryID)
	}
	return ids
}
