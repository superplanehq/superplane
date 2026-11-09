package factories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func TestComputeFactoryVelocityWindows_MatchesDescribeFactoryVelocity(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	repo := "example/repo"
	require.NoError(t, factoryModel.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: &repo}))

	now := time.Now()
	mergedAt := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/1", models.FactoryPullRequestStateMerged, mergedAt)
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/2", models.FactoryPullRequestStateClosed, mergedAt.Add(-time.Hour))
	seedVelocityOrderSpend(t, factoryModel, 3, mergedAt, 2_000_000, 200_000)
	seedSyncedRepositoryMerges(t, r.Organization.ID, factoryModel.ID, repo, repositoryMergeSeed{
		number:   90,
		login:    "ada",
		name:     "Ada Lovelace",
		mergedAt: mergedAt.Add(-2 * time.Hour),
	})

	period := 14
	resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: int32(period),
		Repository: repo,
	})
	require.NoError(t, err)

	windows, err := ComputeFactoryVelocityWindows(db, factoryModel, period, repo, time.Now().In(time.Local))
	require.NoError(t, err)

	assertWindowTotals(t, windows.Current, resp.Totals)
	assertWindowTotals(t, windows.Previous, resp.PreviousTotals)
	assert.Equal(t, resp.HasPreviousWindow, windows.HasPrevious)
	assert.Equal(t, resp.HasPeopleCohort, windows.HasPeople)

	var superplane, people int
	for _, day := range windows.Days {
		superplane += day.SuperplaneMerged
		people += day.PeopleMerged
	}
	assert.Equal(t, windows.Current.SuperplaneMerged, superplane)
	assert.Equal(t, windows.Current.PeopleMerged, people)
	assert.Len(t, windows.Days, period)
}

func TestComputeFactoryVelocityWindows_MultiRepositoryScope(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	primary := "example/repo"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: &primary}))
	for _, repo := range []string{"example/other", "example/readonly"} {
		access := models.FactoryContextRepoAccessWrite
		if repo == "example/readonly" {
			access = models.FactoryContextRepoAccessRead
		}
		_, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{Repository: repo, Access: access})
		require.NoError(t, err)
	}
	now := time.Now()
	for _, repo := range []string{primary, "example/other", "example/readonly"} {
		seedPRArtifact(t, factory, "https://github.com/"+repo+"/pull/1", models.FactoryPullRequestStateMerged, now.Add(-time.Hour))
		for _, at := range []time.Time{now.Add(-time.Hour), now.AddDate(0, 0, -8)} {
			merge := models.NewFactoryVelocityRepositoryMerge(r.Organization.ID, factory.ID, repo, at.Unix(), models.FactoryVelocityMergeSourcePeople, at)
			require.NoError(t, db.Create(&merge).Error)
		}
		sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, now, repo)
		require.NoError(t, err)
		require.NoError(t, sync.RecordSuccess(db, repo, now, now.AddDate(0, 0, -60)))
	}
	aggregate, err := ComputeFactoryVelocityWindows(db, factory, 7, "", now)
	require.NoError(t, err)
	assert.Equal(t, 2, aggregate.Current.SuperplaneMerged)
	assert.Equal(t, 2, aggregate.Current.PeopleMerged)
	assert.Equal(t, 2, aggregate.Previous.PeopleMerged)
	selected, err := loadVelocityWindowData(db, factory, 7, "EXAMPLE/OTHER", now)
	require.NoError(t, err)
	assert.Equal(t, 1, summarizeVelocityWindows(selected).Current.PeopleMerged)
	assert.Equal(t, 1, summarizeVelocityWindows(selected).Previous.PeopleMerged)
	assert.Equal(t, int32(3), aggregateTotals(selected.buckets, true).TasksClosed, "tasks stay workspace-wide, including orders outside the delivery set")
	_, err = ComputeFactoryVelocityWindows(db, factory, 7, "example/readonly", now)
	require.ErrorIs(t, err, models.ErrFactoryPullRequestRepositoryNotInWorkspace)
	require.NoError(t, db.Model(&models.FactoryVelocitySync{}).Where("factory_id = ? AND scope_repository = ?", factory.ID, primary).Update("sync_state", models.FactoryVelocitySyncStateBackfilling).Error)
	aggregate, err = ComputeFactoryVelocityWindows(db, factory, 7, "", now)
	require.NoError(t, err)
	assert.False(t, aggregate.HasPeople)
	selectedWindow, err := ComputeFactoryVelocityWindows(db, factory, 7, "example/other", now)
	require.NoError(t, err)
	assert.True(t, selectedWindow.HasPeople, "a completed selection does not wait for another repository")
}

func assertWindowTotals(t *testing.T, got FactoryVelocityWindowTotals, want *pb.DescribeFactoryVelocityTotals) {
	t.Helper()
	require.NotNil(t, want)
	assert.Equal(t, int(want.GetSuperplaneMerged()), got.SuperplaneMerged)
	assert.Equal(t, int(want.GetPeopleMerged()), got.PeopleMerged)
	assert.Equal(t, int(want.GetWaste()), got.Waste)
	assert.Equal(t, want.GetCostCents(), got.CostCents)
}
