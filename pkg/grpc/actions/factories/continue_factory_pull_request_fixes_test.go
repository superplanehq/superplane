package factories

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

type pausedFixesFixture struct {
	factory     *models.Factory
	pullRequest *models.FactoryPullRequest
	canvas      *models.Canvas
	activity    *models.FactoryPullRequestRun
	rootEvent   *models.CanvasEvent
}

func Test__ContinueFactoryPullRequestFixes(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	db := database.DB(t.Context())

	const staleSHA = "1111111111111111111111111111111111111111"
	const currentSHA = "2222222222222222222222222222222222222222"

	// The checks automation trigger event stores the pull request under data.
	// The title stays stale on purpose: only the head SHA is refreshed.
	triggerData := func(pullRequestID uuid.UUID) map[string]any {
		return map[string]any{
			"data": map[string]any{
				"pullRequest":  map[string]any{"id": pullRequestID.String()},
				"pull_request": map[string]any{"title": "Old title", "head": map[string]any{"sha": staleSHA}},
			},
		}
	}

	pauseFixes := func(t *testing.T, state string) *pausedFixesFixture {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/app",
			Number:     42,
			URL:        "https://github.com/acme/app/pull/42",
			State:      state,
		})
		require.NoError(t, err)

		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "checks")
		handler, err := factory.CreatePRFeedbackHandler(
			db,
			canvas.ID,
			models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
			models.FactoryPRFeedbackHandlerSourcePullRequestChecks,
		)
		require.NoError(t, err)
		require.NoError(t, handler.SetMaximumAttempts(db, 1))

		startRun := func(rootData map[string]any) (*models.CanvasRun, *models.CanvasEvent) {
			rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, prFeedbackPullRequestTriggerNodeID, "default", nil, rootData)
			run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
			require.NoError(t, err)
			return run, rootEvent
		}
		createActivity := func(run *models.CanvasRun) *models.FactoryPullRequestActivityResult {
			result, err := pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
				RunID:             run.ID,
				RevisionSHA:       staleSHA,
				Access:            models.FactoryPullRequestAccessExclusive,
				FeedbackHandlerID: &handler.ID,
			})
			require.NoError(t, err)
			return result
		}

		firstRun, _ := startRun(triggerData(pullRequest.ID))
		first := createActivity(firstRun)
		require.Equal(t, models.FactoryPullRequestActivityOutcomeReady, first.Outcome)
		now := time.Now()
		require.NoError(t, db.Model(firstRun).Updates(map[string]any{
			"state":       models.CanvasRunStateFinished,
			"result":      models.CanvasRunResultFailed,
			"finished_at": now,
		}).Error)
		require.NoError(t, first.Activity.Finalize(db, firstRun))

		limitedRun, rootEvent := startRun(triggerData(pullRequest.ID))
		limited := createActivity(limitedRun)
		require.Equal(t, models.FactoryPullRequestActivityOutcomeLimitReached, limited.Outcome)

		_, err = pullRequest.ObserveRevision(db, currentSHA)
		require.NoError(t, err)

		return &pausedFixesFixture{
			factory:     factory,
			pullRequest: pullRequest,
			canvas:      canvas,
			activity:    limited.Activity,
			rootEvent:   rootEvent,
		}
	}

	continueFixesRequest := func(fixture *pausedFixesFixture) *pb.ContinueFactoryPullRequestFixesRequest {
		return &pb.ContinueFactoryPullRequestFixesRequest{
			FactoryId: fixture.factory.ID.String(),
			PrId:      fixture.pullRequest.ID.String(),
			RunId:     fixture.activity.RunID.String(),
		}
	}

	triggerEvents := func(t *testing.T, fixture *pausedFixesFixture) []models.CanvasEvent {
		t.Helper()
		var events []models.CanvasEvent
		require.NoError(t, db.
			Where("workflow_id = ? AND node_id = ?", fixture.canvas.ID, prFeedbackPullRequestTriggerNodeID).
			Where("execution_id IS NULL").
			Order("created_at ASC").
			Find(&events).Error)
		return events
	}

	requireCode := func(t *testing.T, err error, code codes.Code) {
		t.Helper()
		require.Error(t, err)
		actual, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok, err.Error())
		assert.Equal(t, code, actual)
	}

	t.Run("re-emits the checks trigger event with the current head SHA", func(t *testing.T) {
		fixture := pauseFixes(t, models.FactoryPullRequestStateOpen)

		response, err := ContinueFactoryPullRequestFixes(t.Context(), orgID, continueFixesRequest(fixture))
		require.NoError(t, err)

		events := triggerEvents(t, fixture)
		require.Len(t, events, 3)
		var reemitted *models.CanvasEvent
		for i := range events {
			if events[i].ID.String() == response.GetEventId() {
				reemitted = &events[i]
			}
		}
		require.NotNil(t, reemitted)
		assert.Equal(t, models.CanvasEventStatePending, reemitted.State)
		assert.Equal(t, "default", reemitted.Channel)
		assert.Nil(t, reemitted.ExecutionID)

		data, ok := reemitted.Data.Data().(map[string]any)
		require.True(t, ok)
		payload := data["data"].(map[string]any)
		pullRequestData := payload["pull_request"].(map[string]any)
		assert.Equal(t, currentSHA, pullRequestData["head"].(map[string]any)["sha"])
		assert.Equal(t, "Old title", pullRequestData["title"])
		assert.Equal(t, fixture.pullRequest.ID.String(), payload["pullRequest"].(map[string]any)["id"])

		// The stored event keeps its old SHA, so the run history stays accurate.
		original, err := models.FindCanvasEventForCanvas(db, fixture.canvas.ID, fixture.rootEvent.ID)
		require.NoError(t, err)
		originalHead := original.Data.Data().(map[string]any)["data"].(map[string]any)["pull_request"].(map[string]any)["head"]
		assert.Equal(t, staleSHA, originalHead.(map[string]any)["sha"])

		activity, err := models.FindPullRequestActivityByRunID(db, fixture.activity.RunID)
		require.NoError(t, err)
		assert.NotNil(t, activity.FixesContinuedAt)
	})

	t.Run("rejects a second continue for the same activity", func(t *testing.T) {
		fixture := pauseFixes(t, models.FactoryPullRequestStateOpen)

		_, err := ContinueFactoryPullRequestFixes(t.Context(), orgID, continueFixesRequest(fixture))
		require.NoError(t, err)
		_, err = ContinueFactoryPullRequestFixes(t.Context(), orgID, continueFixesRequest(fixture))

		requireCode(t, err, codes.FailedPrecondition)
		assert.Len(t, triggerEvents(t, fixture), 3)
	})

	// The continue and the new trigger event share one transaction. When the
	// event insert fails, the continue must roll back with it.
	t.Run("rolls back the continue when the event insert fails", func(t *testing.T) {
		fixture := pauseFixes(t, models.FactoryPullRequestStateOpen)

		cause := errors.New("event insert failed")
		const callback = "test:continue_fixes_event_insert_failure"
		require.NoError(t, database.Conn().Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
			table := tx.Statement.Table
			if table == "" && tx.Statement.Schema != nil {
				table = tx.Statement.Schema.Table
			}
			if table != "workflow_events" {
				return
			}
			_ = tx.AddError(cause)
		}))
		t.Cleanup(func() { require.NoError(t, database.Conn().Callback().Create().Remove(callback)) })

		_, err := ContinueFactoryPullRequestFixes(t.Context(), orgID, continueFixesRequest(fixture))

		requireCode(t, err, codes.Internal)
		activity, findErr := models.FindPullRequestActivityByRunID(db, fixture.activity.RunID)
		require.NoError(t, findErr)
		assert.Nil(t, activity.FixesContinuedAt)
		assert.Len(t, triggerEvents(t, fixture), 2)
		var runCount int64
		require.NoError(t, db.Model(&models.CanvasRun{}).Where("workflow_id = ?", fixture.canvas.ID).Count(&runCount).Error)
		assert.Equal(t, int64(2), runCount)
	})

	t.Run("rejects an activity from another organization", func(t *testing.T) {
		fixture := pauseFixes(t, models.FactoryPullRequestStateOpen)
		otherOrg, err := models.CreateOrganization(support.RandomName("other-org"), "")
		require.NoError(t, err)

		_, err = ContinueFactoryPullRequestFixes(t.Context(), otherOrg.ID.String(), continueFixesRequest(fixture))

		requireCode(t, err, codes.NotFound)
		activity, err := models.FindPullRequestActivityByRunID(db, fixture.activity.RunID)
		require.NoError(t, err)
		assert.Nil(t, activity.FixesContinuedAt)
		assert.Len(t, triggerEvents(t, fixture), 2)
	})

	t.Run("rejects a closed pull request", func(t *testing.T) {
		fixture := pauseFixes(t, models.FactoryPullRequestStateOpen)
		require.NoError(t, db.Model(fixture.pullRequest).Update("state", models.FactoryPullRequestStateClosed).Error)

		_, err := ContinueFactoryPullRequestFixes(t.Context(), orgID, continueFixesRequest(fixture))

		requireCode(t, err, codes.FailedPrecondition)
		activity, err := models.FindPullRequestActivityByRunID(db, fixture.activity.RunID)
		require.NoError(t, err)
		assert.Nil(t, activity.FixesContinuedAt)
	})

	t.Run("rejects an activity that is not paused", func(t *testing.T) {
		fixture := pauseFixes(t, models.FactoryPullRequestStateOpen)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return tx.Model(&models.FactoryPullRequestRun{}).
				Where("run_id = ?", fixture.activity.RunID).
				Update("state", models.FactoryPullRequestActivityStateFinished).Error
		}))

		_, err := ContinueFactoryPullRequestFixes(t.Context(), orgID, continueFixesRequest(fixture))

		requireCode(t, err, codes.FailedPrecondition)
		assert.Len(t, triggerEvents(t, fixture), 2)
	})
}
