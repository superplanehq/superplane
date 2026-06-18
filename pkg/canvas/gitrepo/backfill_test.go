package gitrepo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/canvas/gitref"
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
	require.True(t, gitref.IsDraftBranch(draft.GitBranch))

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return gitrepo.BackfillCanvasRepository(ctx, tx, r.GitProvider, r.Organization.ID, canvas.ID)
	}))

	require.True(t, gitref.GitBranchExists(ctx, r.GitProvider, repository.RepoID, models.CanvasGitBranchMain))
	require.True(t, gitref.GitBranchExists(ctx, r.GitProvider, repository.RepoID, draft.GitBranch))

	mainFiles, err := r.GitProvider.ListFiles(ctx, repository.RepoID, models.CanvasGitBranchMain)
	require.NoError(t, err)
	require.Contains(t, mainFiles, gitref.CanvasFileName)
	require.Contains(t, mainFiles, gitref.ConsoleFileName)

	draftFiles, err := r.GitProvider.ListFiles(ctx, repository.RepoID, draft.GitBranch)
	require.NoError(t, err)
	require.Contains(t, draftFiles, gitref.CanvasFileName)
	require.Contains(t, draftFiles, gitref.ConsoleFileName)
}
