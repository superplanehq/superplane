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

func TestDescribeFactoryVelocity_WithoutRepository(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	now := time.Now()
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/1", models.PrArtifactStateMerged, now.Add(-1*time.Hour))
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/2", models.PrArtifactStateMerged, now.Add(-25*time.Hour))
	seedPRArtifact(t, factoryModel, "https://github.com/example/repo/pull/3", models.PrArtifactStateClosed, now.Add(-2*time.Hour))

	resp, err := DescribeFactoryVelocity(ctx, nil, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: 7,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.False(t, resp.HasPeopleCohort, "people cohort must be hidden without a repo")
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
		{"defaults to 7 when zero", 0, 7},
		{"defaults to 7 when negative", -3, 7},
		{"honors 30", 30, 30},
		{"caps to 30", 90, 30},
		{"honors 14", 14, 14},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DescribeFactoryVelocity(ctx, nil, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
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

	_, err := DescribeFactoryVelocity(ctx, nil, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  "00000000-0000-0000-0000-000000000000",
		PeriodDays: 7,
	})
	require.Error(t, err)
}

func TestSubtractSuperPlaneHits(t *testing.T) {
	sp := []prArtifactMeta{
		{url: "https://github.com/example/repo/pull/1", owner: "example", repo: "repo", number: 1},
		{url: "https://github.com/example/repo/pull/2", owner: "example", repo: "repo", number: 2},
	}
	hits := []peopleMerge{
		{url: "https://github.com/example/repo/pull/1", mergedAt: time.Now()},
		{url: "https://github.com/example/repo/pull/2/", mergedAt: time.Now()},
		{url: "https://github.com/example/repo/pull/3", mergedAt: time.Now()},
	}
	got := subtractSuperPlaneHits(hits, sp)
	require.Len(t, got, 1)
	assert.Equal(t, "https://github.com/example/repo/pull/3", got[0].url)
}

func TestParsePRURL(t *testing.T) {
	cases := []struct {
		url    string
		owner  string
		repo   string
		number int
	}{
		{"https://github.com/example/repo/pull/1", "example", "repo", 1},
		{"https://github.com/example/repo/pulls/42", "example", "repo", 42},
		{"https://api.github.com/repos/example/repo/pulls/9", "example", "repo", 9},
		{"https://example.com/", "", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			owner, repo, number := parsePRURL(tc.url)
			assert.Equal(t, tc.owner, owner)
			assert.Equal(t, tc.repo, repo)
			assert.Equal(t, tc.number, number)
		})
	}
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

func TestBuildDayBuckets(t *testing.T) {
	loc := time.Local
	now := time.Date(2026, 8, 17, 15, 4, 5, 0, loc)
	buckets := buildDayBuckets(now, 7)
	require.Len(t, buckets, 7)
	assert.Equal(t, time.Date(2026, 8, 11, 0, 0, 0, 0, loc), buckets[0].start)
	assert.Equal(t, time.Date(2026, 8, 17, 0, 0, 0, 0, loc), buckets[6].start)
	assert.Equal(t, buckets[0].end, buckets[1].start)
	assert.Equal(t, "Tue", dayLabel(buckets[0].start, 7, 0))
	assert.Equal(t, "Mon", dayLabel(buckets[6].start, 7, 6))
}

func TestFillBuckets_IgnoresTimestampsOutsideWindow(t *testing.T) {
	loc := time.Local
	now := time.Date(2026, 8, 17, 15, 0, 0, 0, loc)
	buckets := buildDayBuckets(now, 7)
	merged := now.Add(48 * time.Hour)
	closed := buckets[0].start.Add(-time.Hour)

	fillBuckets(
		buckets,
		[]prArtifactMeta{{mergedAt: &merged, isMerged: true}},
		[]prArtifactMeta{{closedAt: &closed, isClosedNM: true}},
		[]peopleMerge{{mergedAt: merged}},
	)

	for _, bucket := range buckets {
		assert.Zero(t, bucket.superplaneMerged)
		assert.Zero(t, bucket.waste)
		assert.Zero(t, bucket.peopleMerged)
	}
}

func TestAggregateTotals(t *testing.T) {
	buckets := []dayBucket{
		{superplaneMerged: 3, peopleMerged: 1, waste: 1},
		{superplaneMerged: 1, peopleMerged: 3, waste: 0},
	}
	got := aggregateTotals(buckets, true)
	assert.Equal(t, int32(4), got.SuperplaneMerged)
	assert.Equal(t, int32(4), got.PeopleMerged)
	assert.Equal(t, int32(1), got.Waste)
	assert.Equal(t, int32(50), got.SuperplaneSharePct)
	assert.Equal(t, int32(20), got.WastePct, "waste is 1 of 5 SuperPlane closures")

	gotNoPeople := aggregateTotals(buckets, false)
	assert.Equal(t, int32(0), gotNoPeople.PeopleMerged)
	assert.Equal(t, int32(0), gotNoPeople.SuperplaneSharePct)
}

func seedPRArtifact(t *testing.T, factoryModel *models.Factory, url string, state string, at time.Time) *models.FactoryWorkOrderArtifact {
	t.Helper()

	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "PR order", "", nil, nil, nil)
	require.NoError(t, err)

	data := map[string]any{"url": url, "state": state}
	if state == models.PrArtifactStateMerged {
		data["mergedAt"] = at.UTC().Format(time.RFC3339)
	}
	if state == models.PrArtifactStateClosed {
		data["closedAt"] = at.UTC().Format(time.RFC3339)
	}
	artifact, err := order.CreateArtifact(database.DB(t.Context()), models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypePR,
		Data: data,
		Key:  url,
	})
	require.NoError(t, err)
	return artifact
}
