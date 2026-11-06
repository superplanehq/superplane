package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestMergeConfidenceActivityKeepsItsAssessedRevision(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Assess the change", "", &r.User, nil, nil)
	require.NoError(t, err)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "on-pr-risk", Type: models.NodeTypeTrigger,
		Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1)),
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	canvas.FactoryID = &factory.ID
	event := &models.CanvasEvent{WorkflowID: canvas.ID, NodeID: "on-pr-risk", Data: models.NewJSONValue(map[string]any{"data": map[string]any{
		"pullRequest":  map[string]any{"id": pr.ID.String()},
		"pull_request": map[string]any{"head": map[string]any{"sha": "assessed-head"}},
	}})}
	_, err = pr.ObserveRevision(db, "assessed-head")
	require.NoError(t, err)
	for _, head := range []string{"assessed-head", "newer-head"} {
		t.Run(head, func(t *testing.T) {
			current, err := pr.ObserveRevision(db, head)
			require.NoError(t, err)
			run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, event.NodeID, models.CanvasRunStateStarted, "")
			require.NoError(t, err)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return models.MaybeLinkMergeConfidenceRun(tx, canvas, event, run) }))
			linked, err := models.ListPullRequestRuns(db, []uuid.UUID{pr.ID})
			require.NoError(t, err)
			for _, activity := range linked[pr.ID] {
				require.NotNil(t, activity.Revision)
				require.Equal(t, "assessed-head", activity.Revision.SHA)
			}
			require.NoError(t, pr.LockForUpdate(db))
			require.Equal(t, current.Revision.ID, *pr.CurrentRevisionID, "delayed admission must not restore an old head")
		})
	}
}

func TestMergeConfidenceStartInvalidatesTheCurrentHumanReviewDecision(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		previousState string
		provider      string
	}{
		{name: "a new assessment retires the successful decision", previousState: models.FactoryHumanReviewStateDecided},
		{name: "disabled enforcement stays disabled", previousState: models.FactoryHumanReviewStateDisabled},
		{name: "an absent gate stays absent"},
		{name: "Bitbucket skips GitHub check publication", provider: models.FactoryPullRequestProviderBitbucket},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			order, err := factory.CreateWorkOrder(db, "Review the change", "", &r.User, nil, nil)
			require.NoError(t, err)
			pullRequestURL := "https://github.com/acme/app/pull/42"
			if scenario.provider == models.FactoryPullRequestProviderBitbucket {
				pullRequestURL = "https://bitbucket.org/acme/app/pull-requests/42"
			}
			pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{Provider: scenario.provider, URL: pullRequestURL})
			require.NoError(t, err)
			revision, err := pr.ObserveRevision(db, "current-head")
			require.NoError(t, err)
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
				NodeID: "on-pr-risk", Type: models.NodeTypeTrigger,
				Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1)),
			}}, nil)
			require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
			canvas.FactoryID = &factory.ID
			older, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "on-pr-risk", models.CanvasRunStateFinished, models.CanvasRunResultPassed)
			require.NoError(t, err)
			_, err = pr.CreateActivity(db, models.FactoryPullRequestActivityParams{RunID: older.ID, Title: "Merge confidence", Access: models.FactoryPullRequestAccessConcurrent})
			require.NoError(t, err)
			var previous *models.FactoryHumanReviewDecision
			if scenario.previousState != "" {
				previous, err = pr.NewHumanReviewDecision(db, models.FactoryHumanReviewDecision{
					RevisionID: revision.Revision.ID, SourceRunID: &older.ID, AssessedCommit: "current-head", State: scenario.previousState,
					ConfigurationVersion: uuid.NewString(), CheckRunID: 123, CheckURL: "https://github.com/acme/app/runs/123",
					Rationale: "The previous assessment was favorable.", Result: datatypes.JSONMap{"reviewRequired": false},
				})
				require.NoError(t, err)
			}
			run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "on-pr-risk", models.CanvasRunStateStarted, "")
			require.NoError(t, err)
			event := &models.CanvasEvent{WorkflowID: canvas.ID, NodeID: "on-pr-risk", Data: models.NewJSONValue(map[string]any{"data": map[string]any{
				"pullRequest":  map[string]any{"id": pr.ID.String()},
				"pull_request": map[string]any{"head": map[string]any{"sha": "current-head"}},
			}})}
			link := func() {
				t.Helper()
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return models.MaybeLinkMergeConfidenceRun(tx, canvas, event, run) }))
			}
			link()
			link()
			var publications []models.CanvasNodeRequest
			require.NoError(t, db.Where("workflow_id = ? AND type = ?", canvas.ID, models.NodeRequestTypeMergeConfidenceCheck).Find(&publications).Error)
			if scenario.provider == models.FactoryPullRequestProviderBitbucket {
				require.Empty(t, publications)
				return
			}
			require.Len(t, publications, 1, "repeated run admission must keep one durable publication")
			require.Nil(t, publications[0].ExecutionID, "run cancellation must not discard check publication")
			publication := publications[0].Spec.Data().MergeConfidenceCheck
			require.NotNil(t, publication)
			require.Equal(t, pr.ID, publication.PullRequestID)
			require.Equal(t, run.ID, publication.RunID)
			current, err := pr.CurrentHumanReview(db)
			require.NoError(t, err)
			if previous == nil {
				require.Nil(t, current)
				return
			}
			if previous.State == models.FactoryHumanReviewStateDisabled {
				require.Equal(t, previous.ID, current.ID)
				require.Equal(t, models.FactoryHumanReviewStateDisabled, current.State)
				return
			}
			require.NotEqual(t, previous.ID, current.ID)
			require.Equal(t, models.FactoryHumanReviewStateWaitingAssessment, current.State)
			require.Equal(t, run.ID, *current.SourceRunID)
			require.Equal(t, int64(123), current.CheckRunID)
			require.Equal(t, previous.ConfigurationVersion, current.ConfigurationVersion)
			require.Empty(t, current.Result)
			require.True(t, current.PublicationPending)
			retired, err := models.FindFactoryHumanReviewDecision(db, previous.ID)
			require.NoError(t, err)
			require.Equal(t, models.FactoryHumanReviewStateSuperseded, retired.State)
			link()
			repeated, err := pr.CurrentHumanReview(db)
			require.NoError(t, err)
			require.Equal(t, current.ID, repeated.ID)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return models.MaybeLinkMergeConfidenceRun(tx, canvas, event, older) }))
			repeated, err = pr.CurrentHumanReview(db)
			require.NoError(t, err)
			require.Equal(t, current.ID, repeated.ID, "a delayed older hook cannot restore its assessment")
			var count int64
			require.NoError(t, db.Model(&models.FactoryHumanReviewDecision{}).Where("pull_request_id = ?", pr.ID).Count(&count).Error)
			require.EqualValues(t, 2, count)
		})
	}
}

func TestMaybeLinkVisualEvidenceRunShowsTheRunOnThePullRequest(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Show checkout", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: "acme/app",
		Number:     42,
		URL:        "https://github.com/acme/app/pull/42",
		Title:      "Show checkout",
		State:      models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "on-pr-visual-evidence",
		Name:   "On Pull Request",
		Type:   models.NodeTypeTrigger,
		Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(
			models.FactoryAppTemplateVisualEvidenceID,
			1,
		)),
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	require.NoError(t, db.First(canvas, "id = ?", canvas.ID).Error)

	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "on-pr-visual-evidence", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	event := &models.CanvasEvent{
		WorkflowID: canvas.ID,
		NodeID:     "on-pr-visual-evidence",
		Data: models.NewJSONValue(map[string]any{
			"data": map[string]any{
				"pullRequest": map[string]any{"id": pullRequest.ID.String()},
			},
		}),
	}

	require.NoError(t, models.MaybeLinkVisualEvidenceRun(db, canvas, event, run))
	require.NoError(t, models.MaybeLinkVisualEvidenceRun(db, canvas, event, run))

	linked, err := models.ListPullRequestRuns(db, []uuid.UUID{pullRequest.ID})
	require.NoError(t, err)
	require.Len(t, linked[pullRequest.ID], 1)
	assert.Equal(t, run.ID, linked[pullRequest.ID][0].Run.ID)
	assert.Equal(t, "Capture screenshots or a short video", linked[pullRequest.ID][0].Title)
}

func TestMaybeLinkVisualEvidenceRunIgnoresOtherCanvases(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Show checkout", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: "acme/app",
		Number:     7,
		URL:        "https://github.com/acme/app/pull/7",
		State:      models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "on-pr-risk",
		Name:   "On Pull Request",
		Type:   models.NodeTypeTrigger,
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	require.NoError(t, db.First(canvas, "id = ?", canvas.ID).Error)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "on-pr-risk", models.CanvasRunStateStarted, "")
	require.NoError(t, err)

	require.NoError(t, models.MaybeLinkVisualEvidenceRun(db, canvas, &models.CanvasEvent{
		WorkflowID: canvas.ID,
		Data: models.NewJSONValue(map[string]any{
			"data": map[string]any{
				"pullRequest": map[string]any{"id": pullRequest.ID.String()},
			},
		}),
	}, run))

	linked, err := models.ListPullRequestRuns(db, []uuid.UUID{pullRequest.ID})
	require.NoError(t, err)
	assert.Empty(t, linked[pullRequest.ID])
}

func TestMaybeLinkCustomAutomationRunShowsTheRunOnThePullRequest(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Lint the pull request", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: "acme/app",
		Number:     42,
		URL:        "https://github.com/acme/app/pull/42",
		Title:      "Lint the pull request",
		State:      models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "on-pull-request",
		Name:   "On Pull Request",
		Type:   models.NodeTypeTrigger,
		Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(
			models.FactoryAppTemplateCustomAutomationID,
			1,
		)),
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	require.NoError(t, db.Model(canvas).Update("name", "Fix Linter Issues").Error)
	require.NoError(t, db.First(canvas, "id = ?", canvas.ID).Error)

	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "on-pull-request", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	event := &models.CanvasEvent{
		WorkflowID: canvas.ID,
		NodeID:     "on-pull-request",
		Data: models.NewJSONValue(map[string]any{
			"data": map[string]any{
				"pullRequest": map[string]any{"id": pullRequest.ID.String()},
			},
		}),
	}

	require.NoError(t, models.MaybeLinkCustomAutomationRun(db, canvas, event, run))
	require.NoError(t, models.MaybeLinkCustomAutomationRun(db, canvas, event, run))

	linked, err := models.ListPullRequestRuns(db, []uuid.UUID{pullRequest.ID})
	require.NoError(t, err)
	require.Len(t, linked[pullRequest.ID], 1)
	assert.Equal(t, run.ID, linked[pullRequest.ID][0].Run.ID)
	assert.Equal(t, "Fix Linter Issues", linked[pullRequest.ID][0].Title)
}
