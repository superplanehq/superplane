package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestDeleteDraftBranch(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("not found when branch missing from git and db", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "delete-draft-missing")
		_, err := DeleteDraftBranch(
			ctx,
			r.GitProvider,
			r.Organization.ID.String(),
			canvasID,
			"drafts/missing",
		)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("deletes git branch and db metadata", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "delete-draft-both")
		canvasUUID := uuid.MustParse(canvasID)

		createResp, err := CreateDraftBranch(
			ctx,
			r.GitProvider,
			r.Registry,
			r.Organization.ID.String(),
			canvasID,
			"",
		)
		require.NoError(t, err)
		branchName := createResp.GetBranch().GetBranchName()

		repository, err := models.FindRepository(r.Organization.ID, canvasUUID)
		require.NoError(t, err)

		_, err = DeleteDraftBranch(
			ctx,
			r.GitProvider,
			r.Organization.ID.String(),
			canvasID,
			branchName,
		)
		require.NoError(t, err)

		assert.False(t, materialize.GitBranchExists(ctx, r.GitProvider, repository.RepoID, branchName))

		_, err = models.FindDraftBranch(canvasUUID, branchName)
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("cleans stale db metadata when git branch already deleted", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "delete-draft-db-only")
		canvasUUID := uuid.MustParse(canvasID)

		createResp, err := CreateDraftBranch(
			ctx,
			r.GitProvider,
			r.Registry,
			r.Organization.ID.String(),
			canvasID,
			"Stale draft",
		)
		require.NoError(t, err)
		branchName := createResp.GetBranch().GetBranchName()

		repository, err := models.FindRepository(r.Organization.ID, canvasUUID)
		require.NoError(t, err)
		require.NoError(t, r.GitProvider.DeleteBranch(ctx, repository.RepoID, branchName))

		_, err = models.FindDraftBranch(canvasUUID, branchName)
		require.NoError(t, err)

		_, err = DeleteDraftBranch(
			ctx,
			r.GitProvider,
			r.Organization.ID.String(),
			canvasID,
			branchName,
		)
		require.NoError(t, err)

		_, err = models.FindDraftBranch(canvasUUID, branchName)
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("idempotent when called twice", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "delete-draft-idempotent")

		createResp, err := CreateDraftBranch(
			ctx,
			r.GitProvider,
			r.Registry,
			r.Organization.ID.String(),
			canvasID,
			"",
		)
		require.NoError(t, err)
		branchName := createResp.GetBranch().GetBranchName()

		_, err = DeleteDraftBranch(
			ctx,
			r.GitProvider,
			r.Organization.ID.String(),
			canvasID,
			branchName,
		)
		require.NoError(t, err)

		_, err = DeleteDraftBranch(
			ctx,
			r.GitProvider,
			r.Organization.ID.String(),
			canvasID,
			branchName,
		)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})
}
