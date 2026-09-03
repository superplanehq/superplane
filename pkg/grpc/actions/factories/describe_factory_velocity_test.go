package factories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func TestDescribeFactoryVelocity_WithoutRepository(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/1", models.FactoryPullRequestStateMerged, now.Add(-1*time.Hour))
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/2", models.FactoryPullRequestStateMerged, now.Add(-25*time.Hour))
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/3", models.FactoryPullRequestStateClosed, now.Add(-2*time.Hour))

	resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: 7,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.False(t, resp.HasPeopleCohort, "people cohort must be hidden without a repo")
	assert.False(t, resp.PeopleSyncPending, "a workspace with no repository is not waiting for a sync")
	assert.Equal(t, int32(2), resp.Totals.SuperplaneMerged)
	assert.Equal(t, int32(0), resp.Totals.PeopleMerged)
	assert.Equal(t, int32(1), resp.Totals.Waste)
	assert.Equal(t, int32(0), resp.Totals.SuperplaneSharePct, "share should be zero when the people cohort is hidden")
	assert.Len(t, resp.Points, 7)
}

func TestDescribeFactoryVelocity_ClampsPeriodDays(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	for _, tt := range []struct {
		name     string
		input    int32
		expected int
	}{
		{"defaults to the page default when zero", 0, 14},
		{"defaults to the page default when negative", -3, 14},
		{"honors 30", 30, 30},
		{"caps to 30", 90, 30},
		{"honors 14", 14, 14},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
				FactoryId:  factoryModel.ID.String(),
				PeriodDays: tt.input,
			})
			require.NoError(t, err)
			assert.Len(t, resp.Points, tt.expected)
		})
	}
}

func TestDescribeFactoryVelocity_FactoryNotFound(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	_, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  "00000000-0000-0000-0000-000000000000",
		PeriodDays: 7,
	})
	require.Error(t, err)
}

func TestParseOwnerRepo(t *testing.T) {
	owner, repo, ok := parseOwnerRepo("Example/Repo")
	require.True(t, ok)
	assert.Equal(t, "example", owner)
	assert.Equal(t, "repo", repo)

	_, _, ok = parseOwnerRepo("")
	assert.False(t, ok)

	_, _, ok = parseOwnerRepo("just-one-segment")
	assert.False(t, ok)

	_, _, ok = parseOwnerRepo("/foo")
	assert.False(t, ok)
}

func TestDescribeFactoryVelocity_PendingSyncKeepsSuperPlaneCounts(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/1", models.FactoryPullRequestStateMerged, now.Add(-1*time.Hour))

	resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: 7,
		Repository: "example/repo",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.True(t, resp.PeopleSyncPending, "no sync has run for this workspace yet")
	assert.False(t, resp.HasPeopleCohort)
	assert.Nil(t, resp.PeopleSyncedAt)
	assert.Equal(t, int32(1), resp.Totals.SuperplaneMerged)
	assert.Equal(t, int32(0), resp.Totals.PeopleMerged)
	assert.Equal(t, "example/repo", resp.Repository)
}

func TestDescribeFactoryVelocity_CountsSyncedPeopleMerges(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/1", models.FactoryPullRequestStateMerged, now.Add(-1*time.Hour))
	seedSyncedRepositoryMerges(t, r.Organization.ID, factoryModel.ID, "example/repo",
		repositoryMergeSeed{number: 50, login: "octocat", name: "Octo Cat", mergedAt: now.Add(-2 * time.Hour)},
		repositoryMergeSeed{number: 51, login: "octocat", name: "Octo Cat", mergedAt: now.Add(-3 * time.Hour)},
		repositoryMergeSeed{number: 52, login: "hubber", name: "Hub Ber", mergedAt: now.Add(-30 * 24 * time.Hour)},
	)

	resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: 7,
		Repository: "example/repo",
	})
	require.NoError(t, err)

	assert.True(t, resp.HasPeopleCohort)
	assert.False(t, resp.PeopleSyncPending)
	assert.NotNil(t, resp.PeopleSyncedAt)
	assert.Equal(t, int32(1), resp.Totals.SuperplaneMerged)
	assert.Equal(t, int32(2), resp.Totals.PeopleMerged, "the merge outside the window does not count")
	assert.Equal(t, int32(33), resp.Totals.SuperplaneSharePct, "1 of 3 merges came from SuperPlane")

	// The author is not an organization member, so they still earn their own row.
	var octocat *pb.DescribeFactoryVelocityPerson
	for _, person := range resp.People {
		if person.Name == "Octo Cat" {
			octocat = person
		}
	}
	require.NotNil(t, octocat, "a non-member author is still part of what the repository shipped")
	assert.Equal(t, int32(2), octocat.AuthoredMerged)
	assert.Equal(t, int32(0), octocat.FactoryMerged)
}

func TestDescribeFactoryVelocity_CountsAgentMergesAsSuperPlaneOutput(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	// The agent merges of a repository this instance never opened pull requests
	// in. Before the sync classified them, they counted as people output or were
	// dropped, and the SuperPlane series read as zero.
	now := time.Now()
	seedSyncedRepositoryMerges(t, r.Organization.ID, factoryModel.ID, "example/repo",
		repositoryMergeSeed{number: 60, login: "app[bot]", agent: true, mergedAt: now.Add(-2 * time.Hour)},
		repositoryMergeSeed{number: 61, login: "app[bot]", agent: true, mergedAt: now.Add(-3 * time.Hour)},
		repositoryMergeSeed{number: 62, login: "octocat", name: "Octo Cat", mergedAt: now.Add(-4 * time.Hour)},
	)

	resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: 7,
		Repository: "example/repo",
	})
	require.NoError(t, err)

	assert.Equal(t, int32(2), resp.Totals.SuperplaneMerged,
		"agent work counts as SuperPlane output even without a work order behind it")
	assert.Equal(t, int32(1), resp.Totals.PeopleMerged)
	assert.Equal(t, int32(66), resp.Totals.SuperplaneSharePct, "2 of 3 merges came from SuperPlane")

	// The intake breakdown must still add up to the SuperPlane total, so the
	// agent merges appear as automation rather than going missing.
	var automation *pb.DescribeFactoryVelocityIntakeSource
	for _, source := range resp.IntakeSources {
		if source.Key == velocityIntakeKeyAutomation {
			automation = source
		}
	}
	require.NotNil(t, automation, "agent merges have no work order, so they are automation intake")
	assert.Equal(t, int32(2), automation.Merged)

	for _, person := range resp.People {
		assert.NotEqual(t, "app[bot]", person.Name, "agent work is not credited to the app that opened it")
	}
}

// TestDescribeFactoryVelocity_JoinsLinkedGitHubAccount covers the reason linked
// accounts exist: a member whose GitHub login differs from their sign-in
// identity shows up twice in the People table until they link the account.
func TestDescribeFactoryVelocity_JoinsLinkedGitHubAccount(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	seedSyncedRepositoryMerges(t, r.Organization.ID, factoryModel.ID, "example/repo",
		repositoryMergeSeed{number: 70, login: "shiroyasha", name: "Igor", mergedAt: now.Add(-1 * time.Hour)},
		repositoryMergeSeed{number: 71, login: "shiroyasha", name: "Igor", mergedAt: now.Add(-2 * time.Hour)},
	)

	request := &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: 7,
		Repository: "example/repo",
	}

	// The fixture signs in with the GitHub login "testuser", so the author is a
	// stranger and earns a row of their own.
	resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err)
	require.NotNil(t, findVelocityPerson(resp.People, "Igor"),
		"without a link the author is a separate person")
	assert.Nil(t, findVelocityPersonByID(resp.People, r.User.String()),
		"the member has no repository activity of their own yet")

	require.NoError(t, models.SaveAccountLinkedAccount(db, models.NewAccountLinkedAccount(
		r.Account.ID, models.ProviderGitHub, "42", "shiroyasha", "Igor", "https://avatar",
	)))

	resp, err = DescribeFactoryVelocity(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err)

	member := findVelocityPersonByID(resp.People, r.User.String())
	require.NotNil(t, member, "the linked account makes the author the member")
	assert.Equal(t, int32(2), member.AuthoredMerged)
	assert.Nil(t, findVelocityPerson(resp.People, "Igor"),
		"the author must not keep a second row after linking")
}

func findVelocityPerson(people []*pb.DescribeFactoryVelocityPerson, name string) *pb.DescribeFactoryVelocityPerson {
	for _, person := range people {
		if person.Name == name {
			return person
		}
	}
	return nil
}

func findVelocityPersonByID(people []*pb.DescribeFactoryVelocityPerson, id string) *pb.DescribeFactoryVelocityPerson {
	for _, person := range people {
		if person.Id == id {
			return person
		}
	}
	return nil
}

type repositoryMergeSeed struct {
	number int64
	login  string
	name   string
	// agent seeds a merge the SuperPlane agent wrote, which the sync recognizes
	// by its co-author trailer. An empty value seeds a person's merge.
	agent    bool
	mergedAt time.Time
}

func (s repositoryMergeSeed) source() string {
	if s.agent {
		return models.FactoryVelocityMergeSourceAgent
	}
	return models.FactoryVelocityMergeSourcePeople
}

// seedSyncedRepositoryMerges stands in for the velocity sync worker, so handler
// tests exercise the same rows the worker writes.
func seedSyncedRepositoryMerges(
	t *testing.T,
	orgID, factoryID uuid.UUID,
	repository string,
	seeds ...repositoryMergeSeed,
) {
	t.Helper()
	db := database.DB(t.Context())

	merges := make([]models.FactoryVelocityRepositoryMerge, 0, len(seeds))
	for _, seed := range seeds {
		merge := models.NewFactoryVelocityRepositoryMerge(orgID, factoryID, repository, seed.number, seed.source(), seed.mergedAt)
		merge.AuthorLogin = seed.login
		merge.AuthorName = seed.name
		merges = append(merges, merge)
	}

	from := time.Now().AddDate(0, 0, -90)
	to := time.Now().Add(time.Hour)
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, factoryID, from, to, merges))

	sync, err := models.ClaimFactoryVelocitySync(db, factoryID, time.Now())
	require.NoError(t, err)
	require.NotNil(t, sync)
	require.NoError(t, sync.RecordSuccess(db, repository, time.Now(), from))
}

func TestCalendarDayUTCNoon(t *testing.T) {
	loc := time.FixedZone("UTC-3", -3*3600)
	localMidnight := time.Date(2026, 8, 17, 0, 0, 0, 0, loc)
	got := calendarDayUTCNoon(localMidnight)
	assert.True(t, got.Equal(time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)))
}

func TestBuildDayBuckets_CalendarDaysAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	// Monday after the 2026 US spring-forward (Sunday 8 March).
	now := time.Date(2026, 3, 9, 15, 0, 0, 0, loc)
	buckets := buildDayBuckets(now, 7)
	require.Len(t, buckets, 7)

	for i := 1; i < len(buckets); i++ {
		assert.Equal(t, buckets[i-1].end, buckets[i].start)
		assert.Equal(t, 0, buckets[i].start.Hour())
	}
	assert.Equal(t, time.Date(2026, 3, 3, 0, 0, 0, 0, loc), buckets[0].start)
	assert.Equal(t, time.Date(2026, 3, 9, 0, 0, 0, 0, loc), buckets[6].start)
}

func TestBuildDayBuckets(t *testing.T) {
	loc := time.Local
	now := time.Date(2026, 8, 17, 15, 4, 5, 0, loc)
	buckets := buildDayBuckets(now, 7)
	require.Len(t, buckets, 7)
	assert.Equal(t, time.Date(2026, 8, 11, 0, 0, 0, 0, loc), buckets[0].start)
	assert.Equal(t, time.Date(2026, 8, 17, 0, 0, 0, 0, loc), buckets[6].start)
	assert.Equal(t, buckets[0].end, buckets[1].start)
	assert.Equal(t, "Tue 11", dayLabel(buckets[0].start))
	assert.Equal(t, "Mon 17", dayLabel(buckets[6].start))
}

func TestFillBuckets_IgnoresTimestampsOutsideWindow(t *testing.T) {
	loc := time.Local
	now := time.Date(2026, 8, 17, 15, 0, 0, 0, loc)
	buckets := buildDayBuckets(now, 7)
	window := velocityWindow{start: buckets[0].start, end: buckets[len(buckets)-1].end}
	merged := now.Add(48 * time.Hour)
	closed := buckets[0].start.Add(-time.Hour)

	fillBuckets(
		buckets,
		[]models.FactoryVelocityPullRequest{{MergedAt: &merged}, {ClosedAt: &closed}},
		map[uuid.UUID]*velocityOrder{},
		[]models.FactoryVelocityRepositoryMerge{{MergedAt: merged}},
		window,
	)

	for _, bucket := range buckets {
		assert.Zero(t, bucket.superplaneMerged)
		assert.Zero(t, bucket.waste)
		assert.Zero(t, bucket.peopleMerged)
	}
}

func TestFillBuckets_CountsIntakeAndChargesOrdersOnce(t *testing.T) {
	loc := time.Local
	now := time.Date(2026, 8, 17, 15, 0, 0, 0, loc)
	buckets := buildDayBuckets(now, 7)
	window := velocityWindow{start: buckets[0].start, end: buckets[len(buckets)-1].end}

	today := localMidnight(now)
	mergedAt := today.Add(9 * time.Hour)
	orderID := uuid.New()

	// One order that opened two merged pull requests on the same day.
	fillBuckets(
		buckets,
		[]models.FactoryVelocityPullRequest{
			{WorkOrderID: orderID, MergedAt: &mergedAt, IntakeSource: models.FactoryIntakeSourceGitHubIssues},
			{WorkOrderID: orderID, MergedAt: &mergedAt, IntakeSource: models.FactoryIntakeSourceGitHubIssues},
		},
		map[uuid.UUID]*velocityOrder{
			orderID: {id: orderID, day: today, merged: true, costCents: 250, tokens: 1200},
		},
		nil,
		window,
	)

	last := buckets[len(buckets)-1]
	assert.Equal(t, 2, last.superplaneMerged, "both pull requests count as output")
	assert.Equal(t, 2, last.intake[models.FactoryIntakeSourceGitHubIssues])
	assert.Equal(t, int64(250), last.costCents, "the order is charged once")
	assert.Equal(t, int64(1200), last.tokens)
	assert.Equal(t, 1, last.tasksClosed, "two pull requests of one order are one task")
	assert.Zero(t, last.tasksWaste)
	assert.Zero(t, last.wasteCostCents)
}

func TestFillBuckets_ChargesWasteCost(t *testing.T) {
	loc := time.Local
	now := time.Date(2026, 8, 17, 15, 0, 0, 0, loc)
	buckets := buildDayBuckets(now, 7)
	window := velocityWindow{start: buckets[0].start, end: buckets[len(buckets)-1].end}

	today := localMidnight(now)
	orderID := uuid.New()

	fillBuckets(
		buckets,
		nil,
		map[uuid.UUID]*velocityOrder{
			orderID: {id: orderID, day: today, merged: false, costCents: 180},
		},
		nil,
		window,
	)

	last := buckets[len(buckets)-1]
	assert.Equal(t, int64(180), last.costCents)
	assert.Equal(t, int64(180), last.wasteCostCents, "spend on an unmerged close is waste")
	assert.Equal(t, 1, last.tasksClosed)
	assert.Equal(t, 1, last.tasksWaste, "a task that closed without a merge is waste")
}

func TestAggregateTotals(t *testing.T) {
	buckets := []dayBucket{
		{superplaneMerged: 3, peopleMerged: 1, waste: 1, costCents: 400, tokens: 900, wasteCostCents: 100, tasksClosed: 3, tasksWaste: 1},
		{superplaneMerged: 1, peopleMerged: 3, waste: 0, costCents: 200, tokens: 300, tasksClosed: 1},
	}
	got := aggregateTotals(buckets, true)
	assert.Equal(t, int32(4), got.SuperplaneMerged)
	assert.Equal(t, int32(4), got.PeopleMerged)
	assert.Equal(t, int32(1), got.Waste)
	assert.Equal(t, int32(50), got.SuperplaneSharePct)
	assert.Equal(t, int32(20), got.WastePct, "waste is 1 of 5 SuperPlane closures")
	assert.Equal(t, int64(600), got.CostCents)
	assert.Equal(t, int64(1200), got.Tokens)
	assert.Equal(t, int64(100), got.WasteCostCents)
	assert.Equal(t, int32(4), got.TasksClosed)
	assert.Equal(t, int32(1), got.TasksWaste)

	gotNoPeople := aggregateTotals(buckets, false)
	assert.Equal(t, int32(0), gotNoPeople.PeopleMerged)
	assert.Equal(t, int32(0), gotNoPeople.SuperplaneSharePct)
}

func TestHasVelocityOutput(t *testing.T) {
	assert.False(t, hasVelocityOutput(nil))
	assert.False(t, hasVelocityOutput(&pb.DescribeFactoryVelocityTotals{CostCents: 500}),
		"spend alone is not comparable output")
	assert.True(t, hasVelocityOutput(&pb.DescribeFactoryVelocityTotals{Waste: 1}))
	assert.True(t, hasVelocityOutput(&pb.DescribeFactoryVelocityTotals{PeopleMerged: 2}))
}

func TestDayLabel(t *testing.T) {
	loc := time.Local
	start := time.Date(2026, 8, 11, 0, 0, 0, 0, loc)

	assert.Equal(t, "Tue 11", dayLabel(start), "a mid-month day names the weekday and the date")
}

func TestDayLabel_NamesTheMonthWhenTheWindowCrossesIt(t *testing.T) {
	loc := time.Local
	firstOfMonth := time.Date(2026, 9, 1, 0, 0, 0, 0, loc)

	assert.Equal(t, "Tue Sep 1", dayLabel(firstOfMonth),
		"a date in a new month repeats the month so it cannot be misread")

	midMonth := time.Date(2026, 9, 12, 0, 0, 0, 0, loc)
	assert.Equal(t, "Sat 12", dayLabel(midMonth), "days inside one month drop the month")
}

func TestDescribeFactoryVelocity_ComparesPreviousWindow(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	// Two merges in the reported week, one in the week before it.
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/1", models.FactoryPullRequestStateMerged, now.Add(-2*time.Hour))
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/2", models.FactoryPullRequestStateMerged, now.Add(-48*time.Hour))
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/3", models.FactoryPullRequestStateMerged, now.Add(-9*24*time.Hour))

	resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: 7,
	})
	require.NoError(t, err)

	assert.Equal(t, int32(2), resp.Totals.SuperplaneMerged)
	require.NotNil(t, resp.PreviousTotals)
	assert.Equal(t, int32(1), resp.PreviousTotals.SuperplaneMerged)
	assert.True(t, resp.HasPreviousWindow)
}

func TestDescribeFactoryVelocity_ReportsNoComparisonForNewWorkspace(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/1", models.FactoryPullRequestStateMerged, time.Now().Add(-time.Hour))

	resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: 7,
	})
	require.NoError(t, err)

	assert.False(t, resp.HasPreviousWindow, "an empty baseline is not comparable")
	assert.Equal(t, int32(0), resp.PreviousTotals.SuperplaneMerged)
}

func TestDescribeFactoryVelocity_ReportsIntakeAndPeople(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	order, err := factoryModel.CreateWorkOrder(db, "Hand written order", "", &r.User, nil, nil)
	require.NoError(t, err)
	mergedAt := now.Add(-3 * time.Hour)
	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:      "https://github.com/example/repo/pull/7",
		State:    models.FactoryPullRequestStateMerged,
		MergedAt: &mergedAt,
	})
	require.NoError(t, err)

	resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: 7,
	})
	require.NoError(t, err)

	require.Len(t, resp.IntakeSources, 1)
	assert.Equal(t, velocityIntakeKeyManual, resp.IntakeSources[0].Key)
	assert.Equal(t, "Manually created", resp.IntakeSources[0].Label)
	assert.Equal(t, int32(1), resp.IntakeSources[0].Merged)

	require.Len(t, resp.People, 1)
	assert.Equal(t, r.User.String(), resp.People[0].Id)
	assert.Equal(t, int32(1), resp.People[0].FactoryMerged)
	assert.Equal(t, int32(0), resp.People[0].AuthoredMerged)
}

func TestDescribeFactoryVelocity_CountsTasksApartFromPullRequests(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	mergedAt := now.Add(-3 * time.Hour)
	merged, err := factoryModel.CreateWorkOrder(db, "Merged order", "", &r.User, nil, nil)
	require.NoError(t, err)
	for _, url := range []string{
		"https://github.com/example/repo/pull/1",
		"https://github.com/example/repo/pull/2",
	} {
		_, err = merged.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL:      url,
			State:    models.FactoryPullRequestStateMerged,
			MergedAt: &mergedAt,
		})
		require.NoError(t, err)
	}

	closedAt := now.Add(-2 * time.Hour)
	wasted, err := factoryModel.CreateWorkOrder(db, "Closed order", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = wasted.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:      "https://github.com/example/repo/pull/3",
		State:    models.FactoryPullRequestStateClosed,
		ClosedAt: &closedAt,
	})
	require.NoError(t, err)

	resp, err := DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: 7,
	})
	require.NoError(t, err)

	assert.Equal(t, int32(2), resp.Totals.SuperplaneMerged, "both pull requests of the merged order count")
	assert.Equal(t, int32(1), resp.Totals.Waste)
	assert.Equal(t, int32(2), resp.Totals.TasksClosed, "the merged order counts once, however many pull requests it opened")
	assert.Equal(t, int32(1), resp.Totals.TasksWaste)
}

func seedPRArtifact(t *testing.T, factoryModel *models.Factory, url string, state string, at time.Time) *models.FactoryPullRequest {
	t.Helper()

	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "PR order", "", nil, nil, nil)
	require.NoError(t, err)

	params := models.FactoryPullRequestParams{
		URL:   url,
		State: state,
	}
	if state == models.FactoryPullRequestStateMerged && !at.IsZero() {
		params.MergedAt = &at
	}
	if state == models.FactoryPullRequestStateClosed && !at.IsZero() {
		params.ClosedAt = &at
	}
	pullRequest, err := order.CreatePullRequest(database.DB(t.Context()), params)
	require.NoError(t, err)
	return pullRequest
}
