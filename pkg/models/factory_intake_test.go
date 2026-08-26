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

func Test__FactoryIntake(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	t.Run("created intake takes its name from the canvas", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")

		intake, err := factory.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		assert.Equal(t, canvas.ID, intake.CanvasID)
		assert.Equal(t, models.FactoryIntakeSourceGitHubIssues, intake.Source)

		found, err := factory.FindIntake(db, intake.ID)
		require.NoError(t, err)
		assert.Equal(t, canvas.Name, found.Name())
	})

	t.Run("source must be one we know how to run", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Unknown source")

		_, err = factory.CreateIntake(db, canvas.ID, "linear-issues")
		assert.ErrorIs(t, err, models.ErrFactoryIntakeSourceInvalid)
	})

	t.Run("a canvas implements at most one intake", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")

		_, err = factory.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)

		_, err = factory.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceSentryExceptions)
		assert.ErrorIs(t, err, models.ErrFactoryIntakeCanvasInUse)
	})

	t.Run("a factory lists several intakes from the same source", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		first := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")
		second := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues (2)")

		_, err = factory.CreateIntake(db, first.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		_, err = factory.CreateIntake(db, second.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)

		intakes, err := factory.ListIntakes(db)
		require.NoError(t, err)
		require.Len(t, intakes, 2)
		assert.ElementsMatch(t, []string{first.Name, second.Name}, []string{intakes[0].Name(), intakes[1].Name()})
	})

	t.Run("intakes of other factories stay out of the list", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		canvas := support.CreateFactoryCanvas(t, r, other.ID, "GitHub issues")
		_, err = other.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)

		intakes, err := factory.ListIntakes(db)
		require.NoError(t, err)
		assert.Empty(t, intakes)
	})

	t.Run("a soft-deleted canvas hides its intake", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")

		intake, err := factory.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		require.NoError(t, canvas.SoftDeleteInTransaction(db))

		intakes, err := factory.ListIntakes(db)
		require.NoError(t, err)
		assert.Empty(t, intakes)

		_, err = factory.FindIntake(db, intake.ID)
		assert.ErrorIs(t, err, models.ErrFactoryIntakeNotFound)
	})

	t.Run("deleting by canvas clears the row so the canvas can go away", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")

		_, err = factory.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)

		require.NoError(t, models.DeleteFactoryIntakesByCanvas(db, canvas.ID))

		var count int64
		require.NoError(t, db.Model(&models.FactoryIntake{}).Where("canvas_id = ?", canvas.ID).Count(&count).Error)
		assert.Zero(t, count)
	})

	t.Run("finds an intake by canvas", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")
		created, err := factory.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)

		found, err := models.FindFactoryIntakeByCanvasID(db, canvas.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, models.FactoryIntakeSourceGitHubIssues, found.Source)
	})

	t.Run("missing intake reports not found", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, err = factory.FindIntake(db, uuid.New())
		assert.ErrorIs(t, err, models.ErrFactoryIntakeNotFound)
		assert.NotErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
