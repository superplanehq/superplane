package materialize_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func setupReconcileDeletionTestRepository(
	t *testing.T,
	r *support.ResourceRegistry,
	canvasID uuid.UUID,
) string {
	t.Helper()

	repoID := r.GitProvider.GetRepositoryID(provider.RepositoryOptions{
		OrganizationID: r.Organization.ID,
		CanvasID:       canvasID,
	})
	_, err := r.GitProvider.CreateRepository(context.Background(), repoID)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		canvas := &models.Canvas{
			ID:             canvasID,
			OrganizationID: r.Organization.ID,
			Name:           support.RandomName("reconcile-test-canvas"),
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		return canvas.CreatePendingRepositoryInTransaction(tx, r.GitProvider.Name(), repoID)
	}))

	return repoID
}

func TestReconcileDraftBranchDeletionsFromGit(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	canvasID := uuid.New()
	userID := r.User

	repoID := setupReconcileDeletionTestRepository(t, r, canvasID)

	draftBranch := materialize.DefaultDraftBranchName(userID)
	require.NoError(t, r.GitProvider.CreateBranch(ctx, repoID, draftBranch, models.CanvasGitBranchMain))

	headSHA, err := r.GitProvider.Head(ctx, repoID, draftBranch)
	require.NoError(t, err)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.CreateDraftBranchInTransaction(tx, &models.CanvasDraftBranch{
			CanvasID:       canvasID,
			OrganizationID: r.Organization.ID,
			BranchName:     draftBranch,
			DisplayName:    "Draft #1",
			OwnerID:        &userID,
			TipSHA:         headSHA,
		})
	}))

	require.NoError(t, r.GitProvider.DeleteBranch(ctx, repoID, draftBranch))

	t.Run("removes stale db row when git branch is gone", func(t *testing.T) {
		var removed []string
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			var reconcileErr error
			removed, reconcileErr = materialize.ReconcileDraftBranchDeletionsFromGit(
				ctx,
				tx,
				r.GitProvider,
				canvasID,
				materialize.ReconcileDraftBranchDeletionsOptions{},
			)
			return reconcileErr
		})
		require.NoError(t, err)
		require.Equal(t, []string{draftBranch}, removed)

		_, err = models.FindDraftBranch(canvasID, draftBranch)
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("idempotent when already reconciled", func(t *testing.T) {
		var removed []string
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			var reconcileErr error
			removed, reconcileErr = materialize.ReconcileDraftBranchDeletionsFromGit(
				ctx,
				tx,
				r.GitProvider,
				canvasID,
				materialize.ReconcileDraftBranchDeletionsOptions{},
			)
			return reconcileErr
		})
		require.NoError(t, err)
		assert.Empty(t, removed)
	})
}

func TestReconcileDraftBranchDeletionsFromGitKeepsExistingGitBranch(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	canvasID := uuid.New()
	userID := r.User

	repoID := setupReconcileDeletionTestRepository(t, r, canvasID)

	draftBranch := materialize.DefaultDraftBranchName(userID)
	require.NoError(t, r.GitProvider.CreateBranch(ctx, repoID, draftBranch, models.CanvasGitBranchMain))

	headSHA, err := r.GitProvider.Head(ctx, repoID, draftBranch)
	require.NoError(t, err)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.CreateDraftBranchInTransaction(tx, &models.CanvasDraftBranch{
			CanvasID:       canvasID,
			OrganizationID: r.Organization.ID,
			BranchName:     draftBranch,
			DisplayName:    "Draft #1",
			OwnerID:        &userID,
			TipSHA:         headSHA,
		})
	}))

	var removed []string
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var reconcileErr error
		removed, reconcileErr = materialize.ReconcileDraftBranchDeletionsFromGit(
			ctx,
			tx,
			r.GitProvider,
			canvasID,
			materialize.ReconcileDraftBranchDeletionsOptions{BranchName: draftBranch},
		)
		return reconcileErr
	})
	require.NoError(t, err)
	assert.Empty(t, removed)

	branch, err := models.FindDraftBranch(canvasID, draftBranch)
	require.NoError(t, err)
	assert.Equal(t, draftBranch, branch.BranchName)
}
