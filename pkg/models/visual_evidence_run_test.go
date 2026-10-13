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
)

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
