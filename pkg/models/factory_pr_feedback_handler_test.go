package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__FactoryPRFeedbackHandler(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	createHandler := func(t *testing.T, factory *models.Factory, canvasID uuid.UUID) *models.FactoryPRFeedbackHandler {
		t.Helper()
		handler, err := factory.CreatePRFeedbackHandler(
			db,
			canvasID,
			models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
			models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion,
		)
		require.NoError(t, err)
		return handler
	}

	t.Run("created handler takes its name from the canvas", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Address PR feedback")

		handler := createHandler(t, factory, canvas.ID)
		assert.Equal(t, canvas.ID, handler.CanvasID)
		assert.Equal(t, models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, handler.Subject)
		assert.Equal(t, models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion, handler.Source)

		found, err := factory.FindPRFeedbackHandler(db, handler.ID)
		require.NoError(t, err)
		assert.Equal(t, canvas.Name, found.Name())
	})

	t.Run("subject must be one we know how to change", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Unknown subject")

		_, err = factory.CreatePRFeedbackHandler(db, canvas.ID, "gitlab-merge-request", models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion)
		assert.ErrorIs(t, err, models.ErrFactoryPRFeedbackHandlerSubjectInvalid)
	})

	t.Run("source must be one we know how to run", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Unknown source")

		_, err = factory.CreatePRFeedbackHandler(db, canvas.ID, models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, "status-checks")
		assert.ErrorIs(t, err, models.ErrFactoryPRFeedbackHandlerSourceInvalid)
	})

	t.Run("a canvas implements at most one handler", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Address PR feedback")

		_ = createHandler(t, factory, canvas.ID)
		_, err = factory.CreatePRFeedbackHandler(
			db,
			canvas.ID,
			models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
			models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion,
		)
		assert.ErrorIs(t, err, models.ErrFactoryPRFeedbackHandlerCanvasInUse)
	})

	t.Run("a factory lists several handlers from the same source", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		first := support.CreateFactoryCanvas(t, r, factory.ID, "Address PR feedback")
		second := support.CreateFactoryCanvas(t, r, factory.ID, "Address PR feedback (2)")

		_ = createHandler(t, factory, first.ID)
		_ = createHandler(t, factory, second.ID)

		handlers, err := factory.ListPRFeedbackHandlers(db)
		require.NoError(t, err)
		require.Len(t, handlers, 2)
		assert.ElementsMatch(t, []string{first.Name, second.Name}, []string{handlers[0].Name(), handlers[1].Name()})
	})

	t.Run("handlers of other factories stay out of the list", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		canvas := support.CreateFactoryCanvas(t, r, other.ID, "Address PR feedback")
		_ = createHandler(t, other, canvas.ID)

		handlers, err := factory.ListPRFeedbackHandlers(db)
		require.NoError(t, err)
		assert.Empty(t, handlers)
	})

	t.Run("a soft-deleted canvas hides its handler", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Address PR feedback")

		handler := createHandler(t, factory, canvas.ID)
		require.NoError(t, canvas.SoftDeleteInTransaction(db))

		handlers, err := factory.ListPRFeedbackHandlers(db)
		require.NoError(t, err)
		assert.Empty(t, handlers)

		_, err = factory.FindPRFeedbackHandler(db, handler.ID)
		assert.ErrorIs(t, err, models.ErrFactoryPRFeedbackHandlerNotFound)
	})

	t.Run("deleting by canvas clears the row so the canvas can go away", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Address PR feedback")

		_ = createHandler(t, factory, canvas.ID)
		require.NoError(t, models.DeleteFactoryPRFeedbackHandlersByCanvas(db, canvas.ID))

		var count int64
		require.NoError(t, db.Model(&models.FactoryPRFeedbackHandler{}).Where("canvas_id = ?", canvas.ID).Count(&count).Error)
		assert.Zero(t, count)
	})

	t.Run("missing handler reports not found", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, err = factory.FindPRFeedbackHandler(db, uuid.New())
		assert.ErrorIs(t, err, models.ErrFactoryPRFeedbackHandlerNotFound)
		assert.NotErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
