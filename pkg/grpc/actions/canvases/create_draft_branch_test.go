package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestCreateDraftBranch(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := CreateDraftBranch(
			context.Background(),
			r.GitProvider,
			r.Registry,
			r.Organization.ID.String(),
			uuid.New().String(),
			"",
		)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
	})

	t.Run("creates git branch and registers draft metadata via sync", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-sync-canvas")
		canvasUUID := uuid.MustParse(canvasID)
		repository, err := models.FindRepository(r.Organization.ID, canvasUUID)
		require.NoError(t, err)

		response, err := CreateDraftBranch(
			ctx,
			r.GitProvider,
			r.Registry,
			r.Organization.ID.String(),
			canvasID,
			"",
		)
		require.NoError(t, err)

		branch := response.GetBranch()
		require.NotNil(t, branch)
		assert.Equal(t, materialize.DefaultDraftBranchName(r.User), branch.GetBranchName())
		assert.Equal(t, "Draft #1", branch.GetDisplayName())
		assert.NotEmpty(t, branch.GetTipSha())

		assert.True(t, materialize.GitBranchExists(ctx, r.GitProvider, repository.RepoID, branch.GetBranchName()))

		stored, err := models.FindDraftBranch(canvasUUID, branch.GetBranchName())
		require.NoError(t, err)
		assert.Equal(t, "Draft #1", stored.DisplayName)
		assert.Equal(t, branch.GetTipSha(), stored.TipSHA)
	})

	t.Run("uses display name override when provided", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-named-canvas")
		canvasUUID := uuid.MustParse(canvasID)

		response, err := CreateDraftBranch(
			ctx,
			r.GitProvider,
			r.Registry,
			r.Organization.ID.String(),
			canvasID,
			"Release prep",
		)
		require.NoError(t, err)

		branch := response.GetBranch()
		require.NotNil(t, branch)
		assert.Equal(t, "Release prep", branch.GetDisplayName())

		stored, err := models.FindDraftBranch(canvasUUID, branch.GetBranchName())
		require.NoError(t, err)
		assert.Equal(t, "Release prep", stored.DisplayName)
	})

	t.Run("syncs when git branch already exists without metadata", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-git-only-canvas")
		canvasUUID := uuid.MustParse(canvasID)
		repository, err := models.FindRepository(r.Organization.ID, canvasUUID)
		require.NoError(t, err)

		branchName := materialize.DefaultDraftBranchName(r.User)
		require.NoError(t, r.GitProvider.CreateBranch(ctx, repository.RepoID, branchName, models.CanvasGitBranchMain))

		var synced *models.CanvasDraftBranch
		require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
			var syncErr error
			synced, syncErr = materialize.SyncDraftBranchFromGit(
				ctx,
				tx,
				r.GitProvider,
				r.Registry,
				r.Organization.ID,
				canvasUUID,
				branchName,
				materialize.SyncDraftBranchOptions{
					CreatedBy: &r.User,
				},
			)
			return syncErr
		}))

		require.NotNil(t, synced)
		assert.Equal(t, "Draft #1", synced.DisplayName)
		assert.Equal(t, branchName, synced.BranchName)

		response, err := CreateDraftBranch(
			ctx,
			r.GitProvider,
			r.Registry,
			r.Organization.ID.String(),
			canvasID,
			"",
		)
		require.NoError(t, err)
		assert.NotEqual(t, branchName, response.GetBranch().GetBranchName())
	})
}

func TestSyncDraftBranchFromGitRegistersGitOnlyBranch(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	canvasID := createCanvasWithNoopNode(
		authentication.SetUserIdInMetadata(ctx, r.User.String()),
		t,
		r,
		"worker-sync-canvas",
	)
	canvasUUID := uuid.MustParse(canvasID)
	repository, err := models.FindRepository(r.Organization.ID, canvasUUID)
	require.NoError(t, err)

	branchName := materialize.DefaultDraftBranchName(r.User) + "-external"
	require.NoError(t, r.GitProvider.CreateBranch(ctx, repository.RepoID, branchName, models.CanvasGitBranchMain))
	headSHA, err := r.GitProvider.Head(ctx, repository.RepoID, branchName)
	require.NoError(t, err)

	var synced *models.CanvasDraftBranch
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var syncErr error
		synced, syncErr = materialize.SyncDraftBranchFromGit(
			ctx,
			tx,
			r.GitProvider,
			r.Registry,
			r.Organization.ID,
			canvasUUID,
			branchName,
			materialize.SyncDraftBranchOptions{
				HeadSHA: headSHA,
			},
		)
		return syncErr
	}))

	require.NotNil(t, synced)
	assert.Equal(t, "Draft #1", synced.DisplayName)
	assert.Equal(t, headSHA, synced.TipSHA)
	assert.Equal(t, r.User, *synced.OwnerID)

	version, err := models.FindVersionBySHA(canvasUUID, headSHA)
	require.NoError(t, err)
	assert.Equal(t, models.MaterializationStatusReady, version.MaterializationStatus)

	_, err = models.FindDraftBranch(canvasUUID, branchName)
	require.NoError(t, err)
}

func TestSyncDraftBranchFromGitUsesCreatedByForCustomBranchName(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	canvasID := createCanvasWithNoopNode(
		authentication.SetUserIdInMetadata(ctx, r.User.String()),
		t,
		r,
		"worker-sync-custom-branch",
	)
	canvasUUID := uuid.MustParse(canvasID)
	repository, err := models.FindRepository(r.Organization.ID, canvasUUID)
	require.NoError(t, err)

	branchName := "drafts/custom"
	require.NoError(t, r.GitProvider.CreateBranch(ctx, repository.RepoID, branchName, models.CanvasGitBranchMain))
	headSHA, err := r.GitProvider.Head(ctx, repository.RepoID, branchName)
	require.NoError(t, err)

	var synced *models.CanvasDraftBranch
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var syncErr error
		synced, syncErr = materialize.SyncDraftBranchFromGit(
			ctx,
			tx,
			r.GitProvider,
			r.Registry,
			r.Organization.ID,
			canvasUUID,
			branchName,
			materialize.SyncDraftBranchOptions{
				HeadSHA:   headSHA,
				CreatedBy: &r.User,
			},
		)
		return syncErr
	}))

	require.NotNil(t, synced)
	require.NotNil(t, synced.OwnerID)
	assert.Equal(t, r.User, *synced.OwnerID)
	require.NotNil(t, synced.CreatedBy)
	assert.Equal(t, r.User, *synced.CreatedBy)
}
