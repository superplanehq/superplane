package factories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
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

func assertWindowTotals(t *testing.T, got FactoryVelocityWindowTotals, want *pb.DescribeFactoryVelocityTotals) {
	t.Helper()
	require.NotNil(t, want)
	assert.Equal(t, int(want.GetSuperplaneMerged()), got.SuperplaneMerged)
	assert.Equal(t, int(want.GetPeopleMerged()), got.PeopleMerged)
	assert.Equal(t, int(want.GetWaste()), got.Waste)
	assert.Equal(t, want.GetCostCents(), got.CostCents)
}
