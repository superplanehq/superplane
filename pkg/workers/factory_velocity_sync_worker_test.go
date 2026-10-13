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

func Test__VelocitySyncWindow__KeepsCompleteBitbucketHistoryIncremental(t *testing.T) {
	now := time.Now()
	syncedAt := now.Add(-10 * time.Minute)
	backfilledFrom := now.AddDate(0, 0, -velocitySyncBackfillDays-1)

	from := velocitySyncWindow(models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		VCSProvider:      models.ProviderBitbucket,
		SyncedAt:         &syncedAt,
		BackfilledFrom:   &backfilledFrom,
	}, now)

	assert.WithinDuration(t, now.Add(-velocitySyncRecomputeWindow), from, time.Second,
		"a complete Bitbucket history only recomputes recent days")
}

func Test__VelocitySyncWindow__CatchUpReadsTheNewerWindow(t *testing.T) {
	now := time.Now()
	slice := now.Add(-10 * 24 * time.Hour)
	windowStart := velocitySyncBackfillStart(now)
	target := models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		VCSProvider:      models.ProviderBitbucket,
		SyncedAt:         &slice,
		BackfilledFrom:   &windowStart,
		SyncState:        models.FactoryVelocitySyncStateCatchUp,
	}

	from, until := velocitySyncBounds(target, now)
	assert.Equal(t, slice, from, "a long rebuild must catch merges made after the first slice")
	assert.True(t, until.IsZero(), "the catch-up must read the newest pages, not another older slice")

	refreshFrom, refreshUntil := velocityRefreshBounds(target, now)
	assert.Equal(t, from, refreshFrom, "a refresh that finishes the rebuild must not skip the newer window")
	assert.True(t, refreshUntil.IsZero())
}

func Test__BitbucketCatchUpFrom__CoversTheNewestSliceAndTheRecomputeWindow(t *testing.T) {
	now := time.Now()
	oldSlice := now.Add(-10 * 24 * time.Hour)
	target := models.FactoryVelocitySyncTarget{SyncedAt: &oldSlice}
	assert.Equal(t, oldSlice, bitbucketCatchUpFrom(target, now))

	recent := now.Add(-time.Hour)
	target.SyncedAt = &recent
	assert.WithinDuration(t, now.Add(-velocitySyncRecomputeWindow), bitbucketCatchUpFrom(target, now), time.Second,
		"a short rebuild still recomputes the days that can change")
}

func Test__ProgressAfterVelocityRead__ShortBitbucketReadContinuesOlderHistory(t *testing.T) {
	now := time.Now()
	requestedFrom := velocitySyncBackfillStart(now)
	cursor := now.Add(-2 * 24 * time.Hour)
	target := models.FactoryVelocitySyncTarget{VCSProvider: models.ProviderBitbucket}

	replaceFrom, replaceTo, backfilledFrom, state := progressAfterVelocityRead(
		target,
		requestedFrom,
		time.Time{},
		velocityReadCoverage{ContinueBefore: cursor},
		now,
		false,
		now,
	)

	assert.Equal(t, cursor, replaceFrom, "a short read must not delete merges it did not read")
	assert.True(t, replaceTo.After(now), "the first slice must cover merges up to now")
	assert.Equal(t, cursor, backfilledFrom, "the next read must continue strictly before this slice")
	assert.Equal(t, models.FactoryVelocitySyncStateBackfilling, state, "a short read must not mark the rebuild done")
}

func Test__ProgressAfterVelocityRead__ContinuationKeepsTheNewerSlice(t *testing.T) {
	now := time.Now()
	requestedFrom := velocitySyncBackfillStart(now)
	savedSlice := now.Add(-2 * 24 * time.Hour)
	older := now.Add(-10 * 24 * time.Hour)
	target := models.FactoryVelocitySyncTarget{
		VCSProvider:    models.ProviderBitbucket,
		SyncState:      models.FactoryVelocitySyncStateBackfilling,
		BackfilledFrom: &savedSlice,
	}

	replaceFrom, replaceTo, backfilledFrom, state := progressAfterVelocityRead(
		target,
		requestedFrom,
		savedSlice,
		velocityReadCoverage{ContinueBefore: older},
		now,
		false,
		now,
	)

	assert.Equal(t, older, replaceFrom)
	assert.Equal(t, savedSlice, replaceTo, "the older slice must not delete merges already saved")
	assert.Equal(t, older, backfilledFrom)
	assert.Equal(t, models.FactoryVelocitySyncStateBackfilling, state)

	replaceFrom, replaceTo, backfilledFrom, state = progressAfterVelocityRead(
		target,
		requestedFrom,
		savedSlice,
		completeVelocityRead(),
		now,
		false,
		now,
	)
	assert.Equal(t, requestedFrom, replaceFrom)
	assert.Equal(t, savedSlice, replaceTo)
	assert.Equal(t, requestedFrom, backfilledFrom, "finishing the older slice must restore the full history window")
	assert.Equal(t, models.FactoryVelocitySyncStateCatchUp, state,
		"a finished older slice must not mark the rebuild done before the newer window is read")
}

func Test__ProgressAfterVelocityRead__ShortCatchUpDoesNotRestartTheOlderHistory(t *testing.T) {
	now := time.Now()
	windowStart := velocitySyncBackfillStart(now)
	firstSlice := now.Add(-10 * 24 * time.Hour)
	cursor := now.Add(-24 * time.Hour)
	catchUpFrom := firstSlice
	target := models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		VCSProvider:      models.ProviderBitbucket,
		SyncedAt:         &firstSlice,
		BackfilledFrom:   &windowStart,
		SyncState:        models.FactoryVelocitySyncStateCatchUp,
	}

	replaceFrom, replaceTo, backfilledFrom, state := progressAfterVelocityRead(
		target,
		catchUpFrom,
		time.Time{},
		velocityReadCoverage{ContinueBefore: cursor},
		now,
		false,
		now,
	)

	assert.Equal(t, cursor, replaceFrom, "a short catch-up must not delete merges it did not read")
	assert.True(t, replaceTo.After(now))
	assert.Equal(t, cursor, backfilledFrom)
	assert.Equal(t, models.FactoryVelocitySyncStateCatchingUp, state,
		"a short catch-up must not switch back to the older rebuild")

	target.SyncState = models.FactoryVelocitySyncStateCatchingUp
	target.BackfilledFrom = &cursor
	older := now.Add(-3 * 24 * time.Hour)
	replaceFrom, replaceTo, backfilledFrom, state = progressAfterVelocityRead(
		target,
		catchUpFrom,
		cursor,
		velocityReadCoverage{ContinueBefore: older},
		now,
		false,
		now,
	)
	assert.Equal(t, older, replaceFrom)
	assert.Equal(t, cursor, replaceTo, "the next catch-up slice must not delete the newer slice already saved")
	assert.Equal(t, older, backfilledFrom)
	assert.Equal(t, models.FactoryVelocitySyncStateCatchingUp, state)

	replaceFrom, replaceTo, backfilledFrom, state = progressAfterVelocityRead(
		target,
		catchUpFrom,
		cursor,
		completeVelocityRead(),
		now,
		false,
		now,
	)
	assert.Equal(t, catchUpFrom, replaceFrom)
	assert.Equal(t, cursor, replaceTo)
	assert.Equal(t, windowStart, backfilledFrom, "finishing the catch-up must restore the full history window")
	assert.Equal(t, models.FactoryVelocitySyncStateIdle, state,
		"the last catch-up slice must finish instead of starting the newer window again")
}

func Test__ProgressAfterVelocityRead__ShortFullRefreshKeepsTheOriginalStart(t *testing.T) {
	now := time.Now()
	windowStart := now.AddDate(0, 0, -velocitySyncBackfillDays-1)
	syncedAt := now.Add(-time.Hour)
	cursor := now.Add(-15 * 24 * time.Hour)
	target := models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		VCSProvider:      models.ProviderBitbucket,
		SyncedAt:         &syncedAt,
		BackfilledFrom:   &windowStart,
		SyncState:        models.FactoryVelocitySyncStateIdle,
	}

	_, _, backfilledFrom, state := progressAfterVelocityRead(
		target,
		velocitySyncBackfillStart(now),
		time.Time{},
		velocityReadCoverage{ContinueBefore: cursor},
		now,
		false,
		now,
	)

	assert.Equal(t, cursor, backfilledFrom)
	assert.Equal(t, models.FactoryVelocitySyncStateBackfilling, state,
		"a short full refresh must not switch to the seven-day catch-up")

	_, _, _, state = progressAfterVelocityRead(
		target,
		velocitySyncBackfillStart(now),
		time.Time{},
		velocityReadCoverage{ContinueBefore: cursor},
		now.Add(time.Minute),
		false,
		now,
	)
	assert.Equal(t, models.FactoryVelocitySyncStateBackfilling, state,
		"a later store clock must not turn a full refresh into the seven-day catch-up")

	target.SyncState = state
	target.BackfilledFrom = &backfilledFrom
	from, until := velocityRefreshBounds(target, now)
	assert.WithinDuration(t, velocitySyncBackfillStart(now), from, time.Second,
		"the next refresh must keep the original start, not the last seven days")
	assert.Equal(t, cursor, until, "the next refresh must continue strictly before the saved cursor")
	assert.True(t, until.After(from), "the remaining history must still be inside the next read")

	scheduledFrom, scheduledUntil := velocitySyncBounds(target, now)
	assert.Equal(t, from, scheduledFrom, "a later scheduled sync must not shrink the unfinished refresh")
	assert.Equal(t, until, scheduledUntil)
}

func Test__ProgressAfterVelocityRead__ShortRecomputeDoesNotRestartTheOlderHistory(t *testing.T) {
	now := time.Now()
	windowStart := now.AddDate(0, 0, -velocitySyncBackfillDays-1)
	syncedAt := now.Add(-time.Hour)
	cursor := now.Add(-24 * time.Hour)
	target := models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		VCSProvider:      models.ProviderBitbucket,
		SyncedAt:         &syncedAt,
		BackfilledFrom:   &windowStart,
		SyncState:        models.FactoryVelocitySyncStateIdle,
	}

	requestedFrom := now.Add(-velocitySyncRecomputeWindow)
	_, _, backfilledFrom, state := progressAfterVelocityRead(
		target,
		requestedFrom,
		time.Time{},
		velocityReadCoverage{ContinueBefore: cursor},
		now,
		false,
		now,
	)

	assert.Equal(t, cursor, backfilledFrom)
	assert.Equal(t, models.FactoryVelocitySyncStateCatchingUp, state,
		"a busy seven-day recompute must not rebuild the older history")

	_, _, _, state = progressAfterVelocityRead(
		target,
		requestedFrom,
		time.Time{},
		velocityReadCoverage{ContinueBefore: cursor},
		now.Add(time.Minute),
		false,
		now,
	)
	assert.Equal(t, models.FactoryVelocitySyncStateCatchingUp, state,
		"a later store clock must not turn a seven-day recompute into an older rebuild")
}

func Test__VelocitySyncBounds__CatchUpSliceDoesNotRereadOlderHistory(t *testing.T) {
	now := time.Now()
	firstSlice := now.Add(-10 * 24 * time.Hour)
	cursor := now.Add(-24 * time.Hour)
	target := models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		VCSProvider:      models.ProviderBitbucket,
		SyncedAt:         &firstSlice,
		BackfilledFrom:   &cursor,
		SyncState:        models.FactoryVelocitySyncStateCatchingUp,
	}

	from, until := velocitySyncBounds(target, now)
	assert.Equal(t, firstSlice, from, "the next slice must keep the newer window, not the 60-day start")
	assert.Equal(t, cursor, until, "the next slice must continue strictly before the saved catch-up cursor")

	refreshFrom, refreshUntil := velocityRefreshBounds(target, now)
	assert.Equal(t, from, refreshFrom, "a refresh must not restart the older history while the catch-up is open")
	assert.Equal(t, until, refreshUntil)
}

func Test__ProgressAfterVelocityRead__CompleteWindowFinishesTheRebuild(t *testing.T) {
	now := time.Now()
	requestedFrom := velocitySyncBackfillStart(now)
	target := models.FactoryVelocitySyncTarget{VCSProvider: models.ProviderBitbucket}

	replaceFrom, replaceTo, backfilledFrom, state := progressAfterVelocityRead(
		target,
		requestedFrom,
		time.Time{},
		completeVelocityRead(),
		now,
		false,
		now,
	)

	assert.Equal(t, requestedFrom, replaceFrom)
	assert.True(t, replaceTo.After(now))
	assert.Equal(t, requestedFrom, backfilledFrom)
	assert.Equal(t, models.FactoryVelocitySyncStateIdle, state)
}

func Test__BitbucketRebuildUntil__ContinuesOlderThanTheSavedSlice(t *testing.T) {
	now := time.Now()
	syncedAt := now.Add(-time.Minute)
	cursor := now.Add(-2 * 24 * time.Hour)
	target := models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		VCSProvider:      models.ProviderBitbucket,
		SyncedAt:         &syncedAt,
		BackfilledFrom:   &cursor,
		SyncState:        models.FactoryVelocitySyncStateBackfilling,
	}

	until, ok := bitbucketRebuildUntil(target, now)
	require.True(t, ok)
	assert.Equal(t, cursor, until)

	from, bound := velocitySyncBounds(target, now)
	assert.WithinDuration(t, velocitySyncBackfillStart(now), from, time.Second)
	assert.Equal(t, cursor, bound)
}

func Test__BitbucketRebuildUntil__ContinuesAFirstSliceWithoutAFinishedSync(t *testing.T) {
	now := time.Now()
	cursor := now.Add(-2 * 24 * time.Hour)
	target := models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		VCSProvider:      models.ProviderBitbucket,
		BackfilledFrom:   &cursor,
		SyncState:        models.FactoryVelocitySyncStateBackfilling,
	}

	until, ok := bitbucketRebuildUntil(target, now)
	require.True(t, ok)
	assert.Equal(t, cursor, until, "a short first slice must continue even though SyncedAt is still unset")
}

func Test__BitbucketRebuildUntil__DoesNotContinueAFinishedOrMovedRepository(t *testing.T) {
	now := time.Now()
	syncedAt := now.Add(-time.Minute)
	cursor := now.Add(-2 * 24 * time.Hour)
	finished := models.FactoryVelocitySyncTarget{
		Repository:       "example/repo",
		SyncedRepository: "example/repo",
		VCSProvider:      models.ProviderBitbucket,
		SyncedAt:         &syncedAt,
		BackfilledFrom:   &cursor,
		SyncState:        models.FactoryVelocitySyncStateIdle,
	}
	_, ok := bitbucketRebuildUntil(finished, now)
	assert.False(t, ok, "a finished rebuild reads the normal window, not an old cursor")

	moved := finished
	moved.SyncState = models.FactoryVelocitySyncStateBackfilling
	moved.Repository = "example/other"
	_, ok = bitbucketRebuildUntil(moved, now)
	assert.False(t, ok, "a repository change must not continue the previous repository's slice")
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
