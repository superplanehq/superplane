package workers

import (
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func Test__VelocitySyncWindow__BackfillsOnFirstSync(t *testing.T) {
	now := time.Now()
	target := models.FactoryVelocitySyncTarget{Repository: "example/repo"}

	from := velocitySyncWindow(target, now)

	assert.WithinDuration(t, now.AddDate(0, 0, -velocitySyncBackfillDays), from, time.Second,
		"a workspace with no history asks for the whole window, so Velocity is useful at once")
}

func Test__VelocitySyncWindow__RecomputesRecentDaysOnceHistoryIsComplete(t *testing.T) {
	now := time.Now()
	syncedAt := now.Add(-10 * time.Minute)
	backfilledFrom := now.AddDate(0, 0, -velocitySyncBackfillDays-1)

	from := velocitySyncWindow(models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		SyncedAt:         &syncedAt,
		BackfilledFrom:   &backfilledFrom,
	}, now)

	assert.WithinDuration(t, now.Add(-velocitySyncRecomputeWindow), from, time.Second,
		"a complete history only recomputes the days that can still change")
}

func Test__VelocitySyncWindow__KeepsBackfillingAnIncompleteHistory(t *testing.T) {
	now := time.Now()
	syncedAt := now.Add(-10 * time.Minute)
	// A first sync that only reached five days back, for example after an error.
	backfilledFrom := now.AddDate(0, 0, -5)

	from := velocitySyncWindow(models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		SyncedAt:         &syncedAt,
		BackfilledFrom:   &backfilledFrom,
	}, now)

	assert.WithinDuration(t, now.AddDate(0, 0, -velocitySyncBackfillDays), from, time.Second,
		"a short history keeps extending instead of capping itself forever")
}

func Test__VelocitySyncWindow__RestartsWhenTheRepositoryChanges(t *testing.T) {
	now := time.Now()
	syncedAt := now.Add(-10 * time.Minute)
	backfilledFrom := now.AddDate(0, 0, -velocitySyncBackfillDays-1)

	from := velocitySyncWindow(models.FactoryVelocitySyncTarget{
		Repository:       "example/other",
		SyncedRepository: "example/repo",
		SyncedAt:         &syncedAt,
		BackfilledFrom:   &backfilledFrom,
	}, now)

	assert.WithinDuration(t, now.AddDate(0, 0, -velocitySyncBackfillDays), from, time.Second,
		"a new repository has no history, whatever the old one reached")
}

func Test__EarliestBackfill__KeepsTheOldestWindowEverCollected(t *testing.T) {
	now := time.Now()
	backfilledFrom := now.AddDate(0, 0, -velocitySyncBackfillDays)
	target := models.FactoryVelocitySyncTarget{BackfilledFrom: &backfilledFrom}

	got := earliestBackfill(target, now.Add(-velocitySyncRecomputeWindow), false)

	assert.Equal(t, backfilledFrom, got,
		"recomputing a week must not shrink the recorded history to a week")
}

func Test__EarliestBackfill__ForgetsHistoryOfAReplacedRepository(t *testing.T) {
	now := time.Now()
	backfilledFrom := now.AddDate(0, 0, -velocitySyncBackfillDays)
	target := models.FactoryVelocitySyncTarget{BackfilledFrom: &backfilledFrom}

	from := now.AddDate(0, 0, -velocitySyncBackfillDays)
	got := earliestBackfill(target, from, true)

	assert.Equal(t, from, got, "the new repository's history starts where this sync started")
}

func Test__RepositoryMergeRows__ExcludesSuperPlanePullRequests(t *testing.T) {
	target := models.FactoryVelocitySyncTarget{
		OrganizationID: uuid.New(),
		FactoryID:      uuid.New(),
		Repository:     "example/repo",
	}
	now := time.Now()
	people := models.FactoryVelocityMergeSourcePeople

	rows := repositoryMergeRows(target, []repositoryMerge{
		{repository: "example/repo", number: 1, source: people, authorLogin: "octocat", mergedAt: now},
		{repository: "example/repo", number: 2, source: people, authorLogin: "superplane[bot]", mergedAt: now},
		{repository: "example/repo", number: 3, source: people, authorLogin: "hubber", mergedAt: now},
	}, []int64{2})

	require.Len(t, rows, 2, "a pull request this instance opened is already known")
	assert.Equal(t, int64(1), rows[0].Number)
	assert.Equal(t, "octocat", rows[0].AuthorLogin)
	assert.Equal(t, int64(3), rows[1].Number)
	assert.Equal(t, target.FactoryID, rows[0].FactoryID)
	assert.Equal(t, target.OrganizationID, rows[0].OrganizationID)
}

func Test__RepositoryMergeRows__KeepsTheSource(t *testing.T) {
	target := models.FactoryVelocitySyncTarget{
		OrganizationID: uuid.New(),
		FactoryID:      uuid.New(),
		Repository:     "example/repo",
	}
	now := time.Now()

	rows := repositoryMergeRows(target, []repositoryMerge{
		{repository: "example/repo", number: 1, source: models.FactoryVelocityMergeSourceAgent, mergedAt: now},
		{repository: "example/repo", number: 2, source: models.FactoryVelocityMergeSourcePeople, mergedAt: now},
	}, nil)

	require.Len(t, rows, 2)
	assert.True(t, rows[0].IsAgent(), "agent work must survive the write to reach the SuperPlane series")
	assert.False(t, rows[1].IsAgent())
}

func Test__GroupVelocitySyncTargets__SharesOneSearchPerRepository(t *testing.T) {
	integration := uuid.New()
	other := uuid.New()

	groups := groupVelocitySyncTargets([]models.FactoryVelocitySyncTarget{
		{FactoryID: uuid.New(), IntegrationID: integration, Repository: "example/repo"},
		{FactoryID: uuid.New(), IntegrationID: integration, Repository: "Example/Repo"},
		{FactoryID: uuid.New(), IntegrationID: integration, Repository: "example/other"},
		{FactoryID: uuid.New(), IntegrationID: other, Repository: "example/repo"},
	})

	require.Len(t, groups, 3, "workspaces on one repository and integration share a search")
	assert.Len(t, groups[0].targets, 2, "the repository match ignores case")
	assert.Equal(t, "example/other", groups[1].repository)
	assert.Equal(t, other, groups[2].integrationID,
		"a different installation has its own rate limit, so it is its own group")
}

func Test__MergesWithin__KeepsOnlyTheWorkspaceWindow(t *testing.T) {
	now := time.Now()
	merged := []repositoryMerge{
		{number: 1, mergedAt: now.Add(-40 * 24 * time.Hour)},
		{number: 2, mergedAt: now.Add(-2 * time.Hour)},
		{number: 3, mergedAt: now.Add(time.Hour)},
	}

	within := mergesWithin(merged, now.Add(-24*time.Hour), now)

	require.Len(t, within, 1, "a shared search reaches further back than every workspace needs")
	assert.Equal(t, int64(2), within[0].number)
}

func Test__ToRepositoryMerge__KeepsOnlyMergesInsideTheWindow(t *testing.T) {
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now.Add(time.Hour)

	mergedAt := now.Add(-2 * time.Hour)
	merge, ok := toRepositoryMerge(&github.PullRequest{
		Number:   github.Ptr(12),
		MergedAt: &github.Timestamp{Time: mergedAt},
		User:     &github.User{Login: github.Ptr("octocat"), Name: github.Ptr("Octo Cat")},
	}, "example/repo", nil, from, to)
	require.True(t, ok)
	assert.Equal(t, int64(12), merge.number)
	assert.Equal(t, "octocat", merge.authorLogin)
	assert.Equal(t, models.FactoryVelocityMergeSourcePeople, merge.source)
	assert.True(t, merge.mergedAt.Equal(mergedAt), "the merge instant comes from merged_at, not closed_at")

	closedOnly := &github.PullRequest{Number: github.Ptr(13)}
	_, ok = toRepositoryMerge(closedOnly, "example/repo", nil, from, to)
	assert.False(t, ok, "a pull request closed without a merge is not output")

	tooOld := &github.PullRequest{
		Number:   github.Ptr(14),
		MergedAt: &github.Timestamp{Time: from.Add(-time.Hour)},
	}
	_, ok = toRepositoryMerge(tooOld, "example/repo", nil, from, to)
	assert.False(t, ok, "a merge before the window does not belong to it")
}

func Test__ToRepositoryMerge__NamesAgentWorkByItsCoAuthorTrailer(t *testing.T) {
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now.Add(time.Hour)
	mergedAt := &github.Timestamp{Time: now.Add(-2 * time.Hour)}

	// The agent merges through a GitHub App, so the author names the App rather
	// than the agent. Only the trailer on the squashed commit identifies the
	// work, and it does so whichever instance opened the pull request.
	agent := &github.PullRequest{
		Number:         github.Ptr(30),
		MergedAt:       mergedAt,
		MergeCommitSHA: github.Ptr("abc123"),
		User: &github.User{
			Login: github.Ptr("superplane-gh-integration-9000[bot]"),
			Type:  github.Ptr("Bot"),
		},
	}

	merge, ok := toRepositoryMerge(agent, "example/repo", map[string]bool{"abc123": true}, from, to)
	require.True(t, ok, "agent work is SuperPlane output, not something to drop")
	assert.Equal(t, models.FactoryVelocityMergeSourceAgent, merge.source)

	// Without the trailer the same pull request is only an App merge, which is
	// neither SuperPlane output nor work a person wrote.
	_, ok = toRepositoryMerge(agent, "example/repo", map[string]bool{"other": true}, from, to)
	assert.False(t, ok)
}

func Test__HasAgentCoAuthor(t *testing.T) {
	squashed := "feat: Add a thing (#7027)\n\n" +
		"Co-authored-by: SuperPlane Agent <superplaneagent@superplane.com>\n" +
		"Co-authored-by: Igor \u0160ar\u010devi\u0107 <igor@example.com>\n"
	assert.True(t, hasAgentCoAuthor(squashed))

	assert.True(t, hasAgentCoAuthor("fix: thing\n\nCO-AUTHORED-BY: SuperPlane Agent <SuperPlaneAgent@SuperPlane.com>"),
		"a trailer is recognized whatever case it was written in")

	// Another agent tool co-authoring a person's work does not make it SuperPlane
	// output.
	assert.False(t, hasAgentCoAuthor("feat: thing\n\nCo-authored-by: Cursor <cursoragent@cursor.com>"))
	assert.False(t, hasAgentCoAuthor("feat: a plain commit"))
	assert.False(t, hasAgentCoAuthor(""))

	assert.False(t, hasAgentCoAuthor("feat: mentions superplaneagent@superplane.com in the body"),
		"only a trailer counts, so prose naming the agent does not")
}

func Test__ToRepositoryMerge__DropsMachineAuthors(t *testing.T) {
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now.Add(time.Hour)
	mergedAt := &github.Timestamp{Time: now.Add(-2 * time.Hour)}

	// The agent opens pull requests through a GitHub App, so its merges must not
	// count as work the team wrote by hand.
	app := &github.PullRequest{
		Number:   github.Ptr(20),
		MergedAt: mergedAt,
		User: &github.User{
			Login: github.Ptr("superplane-gh-integration-9000[bot]"),
			Type:  github.Ptr("Bot"),
		},
	}
	_, ok := toRepositoryMerge(app, "example/repo", nil, from, to)
	assert.False(t, ok, "a GitHub App merge is not people output")

	untyped := &github.PullRequest{
		Number:   github.Ptr(21),
		MergedAt: mergedAt,
		User:     &github.User{Login: github.Ptr("dependabot[bot]")},
	}
	_, ok = toRepositoryMerge(untyped, "example/repo", nil, from, to)
	assert.False(t, ok, "the login suffix names a bot even when the type is missing")

	person := &github.PullRequest{
		Number:   github.Ptr(22),
		MergedAt: mergedAt,
		User:     &github.User{Login: github.Ptr("octocat"), Type: github.Ptr("User")},
	}
	_, ok = toRepositoryMerge(person, "example/repo", nil, from, to)
	assert.True(t, ok, "a person still counts")
}

// A message the worker cannot read must not be retried forever, and it must not
// be mistaken for a workspace either.
func Test__ConsumeSyncRequested__RejectsMessagesItCannotRead(t *testing.T) {
	w := NewFactoryVelocitySyncWorker("", nil, nil)

	err := w.consumeSyncRequested(tackle.NewFakeDelivery([]byte("not a protobuf message")))
	assert.Error(t, err)

	body, err := proto.Marshal(&pb.FactoryVelocitySyncRequestedMessage{FactoryId: "not-a-uuid"})
	require.NoError(t, err)
	assert.ErrorContains(t, w.consumeSyncRequested(tackle.NewFakeDelivery(body)), "parse factory id")
}

// A requested sync rebuilds the whole report, so a merge whose classification
// changed is corrected instead of keeping what an earlier sync stored.
func Test__VelocitySyncGroup__WindowStart(t *testing.T) {
	now := time.Now()
	syncedAt := now.Add(-10 * time.Minute)
	backfilledFrom := now.AddDate(0, 0, -velocitySyncBackfillDays-1)
	target := models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		SyncedAt:         &syncedAt,
		BackfilledFrom:   &backfilledFrom,
	}

	scheduled := velocitySyncGroup{repository: "example/repo"}
	assert.WithinDuration(t, now.Add(-velocitySyncRecomputeWindow), scheduled.windowStart(target, now), time.Second,
		"a scheduled sync of a complete history only recomputes recent days")

	requested := velocitySyncGroup{repository: "example/repo", fullWindow: true}
	assert.WithinDuration(t, velocitySyncBackfillStart(now), requested.windowStart(target, now), time.Second)
}

// An on-demand sync must not wait out the lease of a scheduled run, or a user
// pressing the button right after a sync would see nothing happen.
func Test__VelocitySyncGroup__ClaimHorizon(t *testing.T) {
	now := time.Now()

	scheduled := velocitySyncGroup{repository: "example/repo"}
	assert.True(t, scheduled.claimHorizon(now).Equal(now.Add(-velocitySyncLease)),
		"a scheduled sync respects the lease of another worker")

	requested := velocitySyncGroup{repository: "example/repo", claimableFrom: now.Add(-velocitySyncOnDemandGuard)}
	assert.True(t, requested.claimHorizon(now).Equal(now.Add(-velocitySyncOnDemandGuard)))
	assert.True(t, requested.claimHorizon(now).After(scheduled.claimHorizon(now)),
		"a requested sync claims a workspace a scheduled sync would still be holding")
}

func Test__SplitOwnerRepo(t *testing.T) {
	owner, repo, ok := splitOwnerRepo(" Example/Repo ")
	require.True(t, ok)
	assert.Equal(t, "example", owner)
	assert.Equal(t, "repo", repo)

	for _, invalid := range []string{"", "one-segment", "/repo", "owner/", "a/b/c"} {
		_, _, ok := splitOwnerRepo(invalid)
		assert.False(t, ok, invalid)
	}
}
