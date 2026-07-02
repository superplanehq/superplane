package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestWorkflowStagingUpsertListAndDiscard(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	var draft *models.CanvasVersion
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		draft, err = models.CreateDraftBranchFromLiveInTransaction(tx, canvas.ID, r.User, "", nil, nil)
		return err
	})
	require.NoError(t, err)

	hasStaging, err := models.HasWorkflowStaging(draft.ID)
	require.NoError(t, err)
	assert.False(t, hasStaging)

	updatedBy := r.User

	row, err := models.UpsertWorkflowStagingPath(
		draft.ID,
		r.Organization.ID,
		"canvas.yaml",
		"nodes: []",
		"",
		&updatedBy,
	)
	require.NoError(t, err)
	assert.Equal(t, "nodes: []", row.Content)
	assert.False(t, row.Deleted)
	assert.Equal(t, draft.ID, row.VersionID)

	hasStaging, err = models.HasWorkflowStaging(draft.ID)
	require.NoError(t, err)
	assert.True(t, hasStaging)

	updated, err := models.UpsertWorkflowStagingPath(
		draft.ID,
		r.Organization.ID,
		"canvas.yaml",
		"nodes: [updated]",
		"",
		&updatedBy,
	)
	require.NoError(t, err)
	assert.Equal(t, "nodes: [updated]", updated.Content)

	_, err = models.UpsertWorkflowStagingPath(
		draft.ID,
		r.Organization.ID,
		"console.yaml",
		"panels: []",
		"",
		&updatedBy,
	)
	require.NoError(t, err)

	rows, err := models.ListWorkflowStaging(draft.ID)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "canvas.yaml", rows[0].Path)
	assert.Equal(t, "console.yaml", rows[1].Path)

	require.NoError(t, models.DiscardWorkflowStaging(draft.ID, []string{"console.yaml"}))

	rows, err = models.ListWorkflowStaging(draft.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "canvas.yaml", rows[0].Path)

	require.NoError(t, models.DiscardWorkflowStaging(draft.ID, nil))

	hasStaging, err = models.HasWorkflowStaging(draft.ID)
	require.NoError(t, err)
	assert.False(t, hasStaging)
}

func TestWorkflowStagingMarkDeleted(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	var draft *models.CanvasVersion
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		draft, err = models.CreateDraftBranchFromLiveInTransaction(tx, canvas.ID, r.User, "", nil, nil)
		return err
	})
	require.NoError(t, err)

	updatedBy := r.User

	require.NoError(t, models.MarkWorkflowStagingPathDeleted(
		draft.ID,
		r.Organization.ID,
		"console.yaml",
		"",
		&updatedBy,
	))

	row, err := models.FindWorkflowStagingPath(draft.ID, "console.yaml")
	require.NoError(t, err)
	assert.True(t, row.Deleted)
	assert.Empty(t, row.Content)

	_, err = models.UpsertWorkflowStagingPath(
		draft.ID,
		r.Organization.ID,
		"console.yaml",
		"panels: [restored]",
		"",
		&updatedBy,
	)
	require.NoError(t, err)

	row, err = models.FindWorkflowStagingPath(draft.ID, "console.yaml")
	require.NoError(t, err)
	assert.False(t, row.Deleted)
	assert.Equal(t, "panels: [restored]", row.Content)
}

func TestWorkflowStagingIsolatedByDraft(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	var firstDraft, secondDraft *models.CanvasVersion
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		firstDraft, err = models.CreateDraftBranchFromLiveInTransaction(tx, canvas.ID, r.User, "", nil, nil)
		if err != nil {
			return err
		}
		secondDraft, err = models.CreateDraftBranchFromLiveInTransaction(tx, canvas.ID, r.User, "", nil, nil)
		return err
	})
	require.NoError(t, err)

	updatedBy := r.User

	_, err = models.UpsertWorkflowStagingPath(
		firstDraft.ID,
		r.Organization.ID,
		"canvas.yaml",
		"draft: first",
		"",
		&updatedBy,
	)
	require.NoError(t, err)

	hasStaging, err := models.HasWorkflowStaging(secondDraft.ID)
	require.NoError(t, err)
	assert.False(t, hasStaging)

	rows, err := models.ListWorkflowStaging(secondDraft.ID)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestWorkflowStagingCascadesOnDraftDelete(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	var draft *models.CanvasVersion
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		draft, err = models.CreateDraftBranchFromLiveInTransaction(tx, canvas.ID, r.User, "", nil, nil)
		return err
	})
	require.NoError(t, err)

	updatedBy := r.User
	_, err = models.UpsertWorkflowStagingPath(
		draft.ID,
		r.Organization.ID,
		"canvas.yaml",
		"nodes: []",
		"",
		&updatedBy,
	)
	require.NoError(t, err)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where("id = ?", draft.ID).
			Where("state = ?", models.CanvasVersionStateDraft).
			Delete(&models.CanvasVersion{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	require.NoError(t, err)

	hasStaging, err := models.HasWorkflowStaging(draft.ID)
	require.NoError(t, err)
	assert.False(t, hasStaging)
}
