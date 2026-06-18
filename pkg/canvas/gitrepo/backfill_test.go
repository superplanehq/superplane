package gitrepo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/canvas/gitrepo"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestBackfillCanvasRepository(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, false)

	draft, err := models.CreateDraftBranchFromLive(canvas.ID, r.User, "Draft #1", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, draft.BranchName)

	// The migration that backfills git_branch from branch_name ships in a
	// separate PR, so emulate its result here: the git_branch-based backfill
	// only acts on draft rows whose git_branch is populated.
	draft.GitBranch = *draft.BranchName
	require.NoError(t, database.Conn().
		Model(&models.CanvasVersion{}).
		Where("id = ?", draft.ID).
		Update("git_branch", draft.GitBranch).Error)
	require.True(t, models.IsDraftBranch(draft.GitBranch))

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return gitrepo.BackfillCanvasRepository(ctx, tx, r.GitProvider, r.Organization.ID, canvas.ID)
	}))

	_, err = r.GitProvider.Head(ctx, repository.RepoID, models.CanvasGitBranchMain)
	require.NoError(t, err)
	_, err = r.GitProvider.Head(ctx, repository.RepoID, draft.GitBranch)
	require.NoError(t, err)

	mainFiles, err := r.GitProvider.ListFiles(ctx, repository.RepoID, models.CanvasGitBranchMain)
	require.NoError(t, err)
	require.Contains(t, mainFiles, models.CanvasFileName)
	require.Contains(t, mainFiles, models.ConsoleFileName)

	draftFiles, err := r.GitProvider.ListFiles(ctx, repository.RepoID, draft.GitBranch)
	require.NoError(t, err)
	require.Contains(t, draftFiles, models.CanvasFileName)
	require.Contains(t, draftFiles, models.ConsoleFileName)
}
