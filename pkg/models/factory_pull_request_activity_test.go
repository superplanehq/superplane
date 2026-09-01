package models_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__FactoryPullRequestActivityCoordination(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	t.Run("observes a revision once and keeps it current", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		sha := activitySHA("aa")

		first, err := pullRequest.ObserveRevision(db, sha)
		require.NoError(t, err)
		require.True(t, first.Current)
		require.NotNil(t, first.Revision)

		second, err := pullRequest.ObserveRevision(db, sha)
		require.NoError(t, err)
		assert.True(t, second.Current)
		assert.Equal(t, first.Revision.ID, second.Revision.ID)
		assert.Equal(t, first.Revision.ID, *reloadPullRequest(t, db, pullRequest).CurrentRevisionID)
		_ = canvas
	})

	t.Run("replaces the current revision without stopping older activities", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		handler := createCheckHandler(t, db, r, pullRequest, canvas, 3)
		oldRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		oldSHA := activitySHA("11")
		newSHA := activitySHA("22")

		_, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             oldRun.ID,
			RevisionSHA:       oldSHA,
			Access:            models.FactoryPullRequestAccessConcurrent,
			FeedbackHandlerID: &handler.ID,
			Description:       "Waiting for checks",
		})
		require.NoError(t, err)

		observed, err := pullRequest.ObserveRevision(db, newSHA)
		require.NoError(t, err)
		require.True(t, observed.Current)
		require.NotEqual(t, oldSHA, observed.Revision.SHA)

		activity, err := models.FindPullRequestActivityByRunID(db, oldRun.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityStateActive, activity.State)
		reloadedRun, err := models.FindUnscopedCanvasRun(db, oldRun.ID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasRunStateStarted, reloadedRun.State)

		reloaded := reloadPullRequest(t, db, pullRequest)
		assert.Equal(t, observed.Revision.ID, *reloaded.CurrentRevisionID)
	})

	t.Run("keeps pull-request-scoped discussion activities across a new revision", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		discussionRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		checkRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		handler := createCheckHandler(t, db, r, pullRequest, canvas, 3)

		discussion, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:       discussionRun.ID,
			Access:      models.FactoryPullRequestAccessExclusive,
			Description: "Address review comment",
		})
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityOutcomeReady, discussion.Outcome)
		assert.Equal(t, models.FactoryPullRequestAccessExclusive, discussion.Activity.Access)

		_, err = pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             checkRun.ID,
			RevisionSHA:       activitySHA("33"),
			Access:            models.FactoryPullRequestAccessConcurrent,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)

		_, err = pullRequest.ObserveRevision(db, activitySHA("44"))
		require.NoError(t, err)

		discussionActivity, err := models.FindPullRequestActivityByRunID(db, discussionRun.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityStateActive, discussionActivity.State)
		assert.Nil(t, discussionActivity.RevisionID)

		checkActivity, err := models.FindPullRequestActivityByRunID(db, checkRun.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityStateActive, checkActivity.State)
	})

	t.Run("rejects a second active activity for the same handler and revision", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		handler := createCheckHandler(t, db, r, pullRequest, canvas, 3)
		firstRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		secondRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		sha := activitySHA("55")

		first, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             firstRun.ID,
			RevisionSHA:       sha,
			Access:            models.FactoryPullRequestAccessConcurrent,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityOutcomeReady, first.Outcome)

		second, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             secondRun.ID,
			RevisionSHA:       sha,
			Access:            models.FactoryPullRequestAccessConcurrent,
			FeedbackHandlerID: &handler.ID,
		})
		require.ErrorIs(t, err, models.ErrFactoryPullRequestActivityDuplicate)
		assert.Nil(t, second)

		_, err = models.FindPullRequestActivityByRunID(db, secondRun.ID)
		assert.ErrorIs(t, err, models.ErrFactoryPullRequestActivityNotFound)
	})

	t.Run("grants exclusive access in request order", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		firstRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		secondRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")

		first, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:  firstRun.ID,
			Access: models.FactoryPullRequestAccessExclusive,
		})
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityOutcomeReady, first.Outcome)
		assert.Equal(t, models.FactoryPullRequestAccessExclusive, first.Activity.Access)

		second, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:  secondRun.ID,
			Access: models.FactoryPullRequestAccessExclusive,
		})
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityOutcomeWaiting, second.Outcome)
		assert.Equal(t, models.FactoryPullRequestAccessWaiting, second.Activity.Access)
		assert.Nil(t, second.Activity.Attempt)

		require.NoError(t, first.Activity.Finalize(db, finishRun(t, db, firstRun, models.CanvasRunResultPassed)))
		assert.Nil(t, reloadPullRequest(t, db, pullRequest).ActiveMutationRunID)

		retry, err := pullRequest.RequestExclusiveAccess(db, second.Activity)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityOutcomeReady, retry.Outcome)
		assert.Equal(t, models.FactoryPullRequestAccessExclusive, retry.Activity.Access)
		assert.Equal(t, secondRun.ID, *reloadPullRequest(t, db, pullRequest).ActiveMutationRunID)
	})

	t.Run("allows concurrent activities without a lease", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		firstRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		secondRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")

		first, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:       firstRun.ID,
			Access:      models.FactoryPullRequestAccessConcurrent,
			RevisionSHA: activitySHA("66"),
		})
		require.NoError(t, err)
		second, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:  secondRun.ID,
			Access: models.FactoryPullRequestAccessConcurrent,
		})
		require.NoError(t, err)

		assert.Equal(t, models.FactoryPullRequestAccessConcurrent, first.Activity.Access)
		assert.Equal(t, models.FactoryPullRequestAccessConcurrent, second.Activity.Access)
		assert.Nil(t, reloadPullRequest(t, db, pullRequest).ActiveMutationRunID)
	})

	t.Run("does not reserve another attempt for the same exclusive activity", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		handler := createCheckHandler(t, db, r, pullRequest, canvas, 3)
		run := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")

		first, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             run.ID,
			RevisionSHA:       activitySHA("77"),
			Access:            models.FactoryPullRequestAccessExclusive,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)
		require.Equal(t, 1, *first.Activity.Attempt)

		retry, err := pullRequest.RequestExclusiveAccess(db, first.Activity)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityOutcomeReady, retry.Outcome)
		assert.Equal(t, 1, *retry.Activity.Attempt)
	})

	t.Run("reserves one attempt under concurrent exclusive requests", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		handler := createCheckHandler(t, db, r, pullRequest, canvas, 3)
		firstRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		secondRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")

		first, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             firstRun.ID,
			RevisionSHA:       activitySHA("88"),
			Access:            models.FactoryPullRequestAccessConcurrent,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)
		second, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             secondRun.ID,
			Access:            models.FactoryPullRequestAccessConcurrent,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)

		type accessOutcome struct {
			result *models.FactoryPullRequestAccessResult
			err    error
		}
		outcomes := make(chan accessOutcome, 2)
		var started sync.WaitGroup
		started.Add(2)

		request := func(activity *models.FactoryPullRequestRun) {
			started.Done()
			started.Wait()
			var result *models.FactoryPullRequestAccessResult
			err := db.Transaction(func(tx *gorm.DB) error {
				var locked models.FactoryPullRequest
				if loadErr := tx.Where("id = ?", pullRequest.ID).First(&locked).Error; loadErr != nil {
					return loadErr
				}
				result, err = locked.RequestExclusiveAccess(tx, activity)
				return err
			})
			outcomes <- accessOutcome{result: result, err: err}
		}

		go request(first.Activity)
		go request(second.Activity)

		firstOutcome := <-outcomes
		secondOutcome := <-outcomes
		require.NoError(t, firstOutcome.err)
		require.NoError(t, secondOutcome.err)

		readyCount := 0
		waitingCount := 0
		attemptCount := 0
		for _, outcome := range []accessOutcome{firstOutcome, secondOutcome} {
			switch outcome.result.Outcome {
			case models.FactoryPullRequestActivityOutcomeReady:
				readyCount++
				require.NotNil(t, outcome.result.Activity.Attempt)
				assert.Equal(t, 1, *outcome.result.Activity.Attempt)
				attemptCount++
			case models.FactoryPullRequestActivityOutcomeWaiting:
				waitingCount++
				assert.Nil(t, outcome.result.Activity.Attempt)
			default:
				t.Fatalf("unexpected outcome %s", outcome.result.Outcome)
			}
		}
		assert.Equal(t, 1, readyCount)
		assert.Equal(t, 1, waitingCount)
		assert.Equal(t, 1, attemptCount)
	})

	t.Run("pauses automatic fixes when the attempt limit is reached", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		handler := createCheckHandler(t, db, r, pullRequest, canvas, 3)

		for i := 1; i <= 3; i++ {
			run := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
			created, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
				RunID:             run.ID,
				RevisionSHA:       activitySHA(string(rune('a' + i))),
				Access:            models.FactoryPullRequestAccessExclusive,
				FeedbackHandlerID: &handler.ID,
			})
			require.NoError(t, err)
			assert.Equal(t, models.FactoryPullRequestActivityOutcomeReady, created.Outcome)
			require.NotNil(t, created.Activity.Attempt)
			assert.Equal(t, i, *created.Activity.Attempt)
			require.NoError(t, created.Activity.Finalize(db, finishRun(t, db, run, models.CanvasRunResultFailed)))
		}

		limitedRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		limited, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             limitedRun.ID,
			RevisionSHA:       activitySHA("ff"),
			Access:            models.FactoryPullRequestAccessExclusive,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityOutcomeLimitReached, limited.Outcome)
		assert.Equal(t, models.FactoryPullRequestActivityStateLimitReached, limited.Activity.State)
		assert.Nil(t, limited.Activity.Attempt)
		require.NotNil(t, limited.Activity.AttemptLimit)
		assert.Equal(t, 3, *limited.Activity.AttemptLimit)
		assert.Nil(t, reloadPullRequest(t, db, pullRequest).ActiveMutationRunID)
	})

	t.Run("resets attempts after a passed evaluation", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		handler := createCheckHandler(t, db, r, pullRequest, canvas, 3)

		failed := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		failedActivity, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             failed.ID,
			RevisionSHA:       activitySHA("b1"),
			Access:            models.FactoryPullRequestAccessExclusive,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, *failedActivity.Activity.Attempt)
		require.NoError(t, failedActivity.Activity.Finalize(db, finishRun(t, db, failed, models.CanvasRunResultFailed)))

		passed := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		passedActivity, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             passed.ID,
			RevisionSHA:       activitySHA("b2"),
			Access:            models.FactoryPullRequestAccessConcurrent,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)
		assert.Nil(t, passedActivity.Activity.Attempt)
		require.NoError(t, passedActivity.Activity.Finalize(db, finishRun(t, db, passed, models.CanvasRunResultPassed)))

		next := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		nextActivity, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             next.ID,
			RevisionSHA:       activitySHA("b3"),
			Access:            models.FactoryPullRequestAccessExclusive,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, nextActivity.Activity.Attempt)
		assert.Equal(t, 1, *nextActivity.Activity.Attempt)
	})

	t.Run("releases the lease after pass, failure, and cancellation", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)

		for _, result := range []string{models.CanvasRunResultPassed, models.CanvasRunResultFailed, models.CanvasRunResultCancelled} {
			run := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
			created, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
				RunID:       run.ID,
				Access:      models.FactoryPullRequestAccessExclusive,
				Description: result,
			})
			require.NoError(t, err)
			require.NotNil(t, reloadPullRequest(t, db, pullRequest).ActiveMutationRunID)

			finished := finishRun(t, db, run, result)
			require.NoError(t, created.Activity.Finalize(db, finished))

			activity, err := models.FindPullRequestActivityByRunID(db, run.ID)
			require.NoError(t, err)
			assert.Equal(t, models.FactoryPullRequestAccessReleased, activity.Access)
			assert.Equal(t, models.FactoryPullRequestActivityStateFinished, activity.State)
			assert.Nil(t, reloadPullRequest(t, db, pullRequest).ActiveMutationRunID)
		}
	})

	t.Run("finishes a cancelled revision-bound activity", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		handler := createCheckHandler(t, db, r, pullRequest, canvas, 3)
		run := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")

		created, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             run.ID,
			RevisionSHA:       activitySHA("c1"),
			Access:            models.FactoryPullRequestAccessExclusive,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)

		require.NoError(t, created.Activity.Finalize(db, finishRun(t, db, run, models.CanvasRunResultCancelled)))
		activity, err := models.FindPullRequestActivityByRunID(db, run.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityStateFinished, activity.State)
		assert.Equal(t, models.FactoryPullRequestAccessReleased, activity.Access)
	})

	t.Run("grants exclusive access after the current revision changes", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		handler := createCheckHandler(t, db, r, pullRequest, canvas, 3)
		run := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")

		created, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             run.ID,
			RevisionSHA:       activitySHA("c2"),
			Access:            models.FactoryPullRequestAccessConcurrent,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)

		_, err = pullRequest.ObserveRevision(db, activitySHA("c3"))
		require.NoError(t, err)

		granted, err := pullRequest.RequestExclusiveAccess(db, created.Activity)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestActivityOutcomeReady, granted.Outcome)
		assert.Equal(t, models.FactoryPullRequestAccessExclusive, granted.Activity.Access)
		assert.Equal(t, models.FactoryPullRequestActivityStateActive, granted.Activity.State)
	})

	t.Run("updates a description after the attempt limit", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		handler := createCheckHandler(t, db, r, pullRequest, canvas, 1)
		first := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		created, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             first.ID,
			RevisionSHA:       activitySHA("d1"),
			Access:            models.FactoryPullRequestAccessExclusive,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)
		require.NoError(t, created.Activity.Finalize(db, finishRun(t, db, first, models.CanvasRunResultFailed)))

		limitedRun := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		limited, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             limitedRun.ID,
			RevisionSHA:       activitySHA("d2"),
			Access:            models.FactoryPullRequestAccessExclusive,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)
		require.Equal(t, models.FactoryPullRequestActivityOutcomeLimitReached, limited.Outcome)
		require.NoError(t, limited.Activity.UpdateDescription(db, "Automatic fixes paused after 1 attempts"))

		reloaded, err := models.FindPullRequestActivityByRunID(db, limitedRun.ID)
		require.NoError(t, err)
		assert.Equal(t, "Automatic fixes paused after 1 attempts", reloaded.Description)
		assert.Equal(t, models.FactoryPullRequestActivityStateLimitReached, reloaded.State)
	})

	t.Run("clears the lease and activity when a run is deleted", func(t *testing.T) {
		pullRequest, canvas := createActivityFixture(t, db, r)
		run := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		_, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:  run.ID,
			Access: models.FactoryPullRequestAccessExclusive,
		})
		require.NoError(t, err)
		require.NotNil(t, reloadPullRequest(t, db, pullRequest).ActiveMutationRunID)

		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			_, err := run.DeleteChain(tx)
			return err
		}))

		assert.Nil(t, reloadPullRequest(t, db, pullRequest).ActiveMutationRunID)
		_, err = models.FindPullRequestActivityByRunID(db, run.ID)
		assert.ErrorIs(t, err, models.ErrFactoryPullRequestActivityNotFound)
	})

	t.Run("factory cleanup removes revisions after clearing the current pointer", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order, err := factoryModel.CreateWorkOrder(db, "PR order", "", &r.User, nil, nil)
		require.NoError(t, err)
		pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL: "https://github.com/acme/app/pull/901",
		})
		require.NoError(t, err)
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
			nil,
		)
		run := createCanvasRun(t, db, canvas.ID, models.CanvasRunStateStarted, "")
		_, err = pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:       run.ID,
			RevisionSHA: activitySHA("e1"),
			Access:      models.FactoryPullRequestAccessConcurrent,
		})
		require.NoError(t, err)
		require.NoError(t, factoryModel.SoftDelete(db))

		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			_, complete, cleanErr := models.NewFactoryResourceCleaner(tx, factoryModel).WithLimit(500).Run()
			require.NoError(t, cleanErr)
			assert.True(t, complete)
			return nil
		}))

		var revisionCount int64
		require.NoError(t, db.Model(&models.FactoryPullRequestRevision{}).
			Where("pull_request_id = ?", pullRequest.ID).
			Count(&revisionCount).Error)
		assert.Zero(t, revisionCount)
		var pullRequestCount int64
		require.NoError(t, db.Model(&models.FactoryPullRequest{}).Where("id = ?", pullRequest.ID).Count(&pullRequestCount).Error)
		assert.Zero(t, pullRequestCount)
	})
}

func Test__FindPRFeedbackHandlerByCanvasID(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factoryModel.ID, "Address PR feedback")

	missing, err := models.FindPRFeedbackHandlerByCanvasID(db, canvas.ID)
	require.NoError(t, err)
	assert.Nil(t, missing)

	handler, err := factoryModel.CreatePRFeedbackHandler(
		db,
		canvas.ID,
		models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
		models.FactoryPRFeedbackHandlerSourcePullRequestChecks,
	)
	require.NoError(t, err)
	require.NoError(t, handler.SetMaximumAttempts(db, 3))

	found, err := models.FindPRFeedbackHandlerByCanvasID(db, canvas.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, handler.ID, found.ID)
	require.NotNil(t, found.MaximumAttempts)
	assert.Equal(t, 3, *found.MaximumAttempts)
}

func createActivityFixture(t *testing.T, db *gorm.DB, r *support.ResourceRegistry) (*models.FactoryPullRequest, *models.Canvas) {
	t.Helper()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "PR order", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: "https://github.com/acme/app/pull/41",
	})
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		nil,
	)
	return pullRequest, canvas
}

func createCheckHandler(
	t *testing.T,
	db *gorm.DB,
	r *support.ResourceRegistry,
	pullRequest *models.FactoryPullRequest,
	canvas *models.Canvas,
	maximumAttempts int,
) *models.FactoryPRFeedbackHandler {
	t.Helper()
	factoryModel, err := models.FindFactory(db, pullRequest.OrganizationID, pullRequest.FactoryID)
	require.NoError(t, err)
	handlerCanvas := support.CreateFactoryCanvas(t, r, factoryModel.ID, support.RandomName("checks"))
	handler, err := factoryModel.CreatePRFeedbackHandler(
		db,
		handlerCanvas.ID,
		models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
		models.FactoryPRFeedbackHandlerSourcePullRequestChecks,
	)
	require.NoError(t, err)
	require.NoError(t, handler.SetMaximumAttempts(db, maximumAttempts))
	_ = canvas
	return handler
}

func createCanvasRun(t *testing.T, db *gorm.DB, canvasID uuid.UUID, state, result string) *models.CanvasRun {
	t.Helper()
	run, err := models.CreateCanvasRunInTransaction(db, canvasID, "trigger", state, result)
	require.NoError(t, err)
	return run
}

func finishRun(t *testing.T, db *gorm.DB, run *models.CanvasRun, result string) *models.CanvasRun {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Model(run).Updates(map[string]any{
		"state":       models.CanvasRunStateFinished,
		"result":      result,
		"finished_at": now,
		"updated_at":  now,
	}).Error)
	run.State = models.CanvasRunStateFinished
	run.Result = result
	run.FinishedAt = &now
	return run
}

func reloadPullRequest(t *testing.T, db *gorm.DB, pullRequest *models.FactoryPullRequest) *models.FactoryPullRequest {
	t.Helper()
	var reloaded models.FactoryPullRequest
	require.NoError(t, db.Where("id = ?", pullRequest.ID).First(&reloaded).Error)
	return &reloaded
}

func activitySHA(prefix string) string {
	const hex = "0123456789abcdef"
	sha := prefix
	for len(sha) < 40 {
		sha += string(hex[len(sha)%len(hex)])
	}
	return sha
}
