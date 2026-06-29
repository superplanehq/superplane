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

func TestNextDraftDisplayNameUsesMonotonicCanvasCounter(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		first, err := models.NextDraftDisplayNameInTransaction(tx, canvas.ID)
		if err != nil {
			return err
		}
		assert.Equal(t, "Draft #1", first)

		second, err := models.NextDraftDisplayNameInTransaction(tx, canvas.ID)
		if err != nil {
			return err
		}
		assert.Equal(t, "Draft #2", second)

		return nil
	})
	require.NoError(t, err)

	var counter int
	require.NoError(t, database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", canvas.ID).
		Pluck("next_draft_display_number", &counter).
		Error)
	assert.Equal(t, 3, counter)
}

func TestNextDraftDisplayNameDoesNotReuseNumbersAfterDelete(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	var middleBranchName string
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		first, err := models.CreateDraftBranchFromLiveInTransaction(
			tx, canvas.ID, r.User, "", nil, nil,
		)
		if err != nil {
			return err
		}
		assert.Equal(t, "Draft #1", first.DisplayName)

		second, err := models.CreateDraftBranchFromLiveInTransaction(
			tx, canvas.ID, r.User, "", nil, nil,
		)
		if err != nil {
			return err
		}
		assert.Equal(t, "Draft #2", second.DisplayName)

		third, err := models.CreateDraftBranchFromLiveInTransaction(
			tx, canvas.ID, r.User, "", nil, nil,
		)
		if err != nil {
			return err
		}
		assert.Equal(t, "Draft #3", third.DisplayName)
		require.NotEmpty(t, second.GitBranch)
		middleBranchName = second.GitBranch

		return deleteRegisteredDraftBranchInTransaction(tx, canvas.ID, middleBranchName)
	})
	require.NoError(t, err)

	var nextBranch *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		nextBranch, err = models.CreateDraftBranchFromLiveInTransaction(
			tx, canvas.ID, r.User, "", nil, nil,
		)
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, "Draft #4", nextBranch.DisplayName)
}

func deleteRegisteredDraftBranchInTransaction(tx *gorm.DB, canvasID uuid.UUID, branchName string) error {
	result := tx.
		Where("workflow_id = ?", canvasID).
		Where("git_branch = ?", branchName).
		Where("state = ?", models.CanvasVersionStateDraft).
		Delete(&models.CanvasVersion{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
