package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestDeleteWorkflowBranchCascadesCommits(t *testing.T) {
	r := support.Setup(t)

	canvasID := uuid.New()
	ownerID := r.User
	now := time.Now()

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		canvas := models.Canvas{
			ID:             canvasID,
			OrganizationID: r.Organization.ID,
			Name:           "branch-delete-test",
			CreatedBy:      &ownerID,
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}
		if err := tx.Create(&canvas).Error; err != nil {
			return err
		}

		mainVersion, _, err := models.CreateInitialCommitInTransaction(
			tx,
			canvasID,
			ownerID,
			models.CanvasGitBranchMain,
			"Initialize canvas",
			nil,
			nil,
		)
		if err != nil {
			return err
		}

		if err := tx.Model(&models.Canvas{}).
			Where("id = ?", canvasID).
			Update("live_version_id", mainVersion.ID).Error; err != nil {
			return err
		}

		featureBranch, err := models.CreateWorkflowBranch(tx, canvasID, "feat/delete-me", &mainVersion.ID)
		if err != nil {
			return err
		}

		_, err = models.CreateCommitOnBranch(tx, models.CreateCommitInput{
			WorkflowID:    canvasID,
			BranchName:    featureBranch.Name,
			OwnerID:       ownerID,
			CommitMessage: "Feature commit",
			Nodes:         append([]models.Node(nil), mainVersion.Nodes...),
			Edges:         append([]models.Edge(nil), mainVersion.Edges...),
		})
		if err != nil {
			return err
		}

		count, err := models.CountBranchCommitsInTransaction(tx, canvasID, featureBranch.Name)
		if err != nil {
			return err
		}
		require.Equal(t, int64(1), count)

		return models.DeleteWorkflowBranch(tx, canvasID, featureBranch.Name)
	})
	require.NoError(t, err)

	_, err = models.FindWorkflowBranch(database.Conn(), canvasID, "feat/delete-me")
	require.Error(t, err)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	count, err := models.CountBranchCommitsInTransaction(database.Conn(), canvasID, "feat/delete-me")
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	mainBranch, err := models.FindMainWorkflowBranch(database.Conn(), canvasID)
	require.NoError(t, err)
	require.NotNil(t, mainBranch.HeadVersionID)
}
