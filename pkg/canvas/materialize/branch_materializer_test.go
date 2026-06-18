package materialize_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func newBranchMaterializer(r *support.ResourceRegistry) *materialize.BranchMaterializer {
	return &materialize.BranchMaterializer{
		GitProvider: r.GitProvider,
		Registry:    r.Registry,
		Encryptor:   r.Encryptor,
		AuthService: r.AuthService,
	}
}

func seedMainBranch(t *testing.T, ctx context.Context, r *support.ResourceRegistry, repository *models.Repository, name string) string {
	t.Helper()

	canvasYAML := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: ` + name + `
spec:
  nodes: []
  edges: []
`)

	consoleYAML, err := models.CanvasVersionToConsoleYML(&models.CanvasVersion{
		WorkflowID: repository.CanvasID,
		Name:       name,
	})
	require.NoError(t, err)

	_, err = r.GitProvider.CreateRepository(ctx, repository.RepoID)
	if err != nil && err != git.ErrInvalidRepositoryID {
		require.NoError(t, err)
	}

	sha, err := r.GitProvider.Commit(ctx, repository.RepoID, git.CommitOptions{
		Branch:  models.CanvasGitBranchMain,
		Message: "Initial canvas",
		Author:  git.CommitAuthor{Name: "tester", Email: "tester@example.com"},
		Operations: []git.FileOperation{
			{Path: models.CanvasFileName, Content: bytes.NewReader(canvasYAML), SizeBytes: int64(len(canvasYAML))},
			{Path: models.ConsoleFileName, Content: bytes.NewReader(consoleYAML), SizeBytes: int64(len(consoleYAML))},
		},
	})
	require.NoError(t, err)
	return sha
}

func TestBranchMaterializer_MaterializeLive(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, false)

	headSHA := seedMainBranch(t, ctx, r, repository, canvas.Name)

	m := newBranchMaterializer(r)
	require.NoError(t, m.MaterializeBranch(ctx, canvas.ID, models.CanvasGitBranchMain, headSHA, nil))

	live, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)
	require.Equal(t, headSHA, live.CommitSHA)
	require.Equal(t, models.CanvasGitBranchMain, live.GitBranch)
	require.Equal(t, models.MaterializationStatusReady, live.MaterializationStatus)

	// Re-running with the same head is a no-op and keeps the same live version.
	require.NoError(t, m.MaterializeBranch(ctx, canvas.ID, models.CanvasGitBranchMain, headSHA, nil))
	live2, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)
	require.Equal(t, live.ID, live2.ID)
}

func TestBranchMaterializer_SkipsStaleLiveNotification(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, false)

	staleSHA := seedMainBranch(t, ctx, r, repository, canvas.Name)

	updated := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: ` + canvas.Name + `
  description: updated
spec:
  nodes: []
  edges: []
`)

	newHead, err := r.GitProvider.Commit(ctx, repository.RepoID, git.CommitOptions{
		Branch:          models.CanvasGitBranchMain,
		BaseBranch:      models.CanvasGitBranchMain,
		ExpectedHeadSHA: staleSHA,
		Message:         "update canvas",
		Author:          git.CommitAuthor{Name: "tester", Email: "tester@example.com"},
		Operations: []git.FileOperation{
			{Path: models.CanvasFileName, Content: bytes.NewReader(updated), SizeBytes: int64(len(updated))},
		},
	})
	require.NoError(t, err)
	require.NotEqual(t, staleSHA, newHead)

	m := newBranchMaterializer(r)
	require.NoError(t, m.MaterializeBranch(ctx, canvas.ID, models.CanvasGitBranchMain, staleSHA, nil))

	_, err = models.FindVersionByCommitSHA(canvas.ID, staleSHA)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestBranchMaterializer_MaterializeDraft(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, false)

	mainSHA := seedMainBranch(t, ctx, r, repository, canvas.Name)

	draftBranch := models.NewDraftBranchName()
	require.NoError(t, r.GitProvider.CreateBranch(ctx, repository.RepoID, draftBranch, models.CanvasGitBranchMain))
	draftHead, err := r.GitProvider.Head(ctx, repository.RepoID, draftBranch)
	require.NoError(t, err)
	require.Equal(t, mainSHA, draftHead)

	m := newBranchMaterializer(r)
	require.NoError(t, m.MaterializeBranch(ctx, canvas.ID, draftBranch, draftHead, &r.User))

	draft, err := models.FindDraftVersionByBranch(canvas.ID, draftBranch)
	require.NoError(t, err)
	require.Equal(t, draftBranch, draft.GitBranch)
	require.Equal(t, draftHead, draft.CommitSHA)
	require.Equal(t, models.MaterializationStatusReady, draft.MaterializationStatus)
	require.NotNil(t, draft.OwnerID)
	require.Equal(t, r.User, *draft.OwnerID)
}

func TestBranchMaterializer_SweepsDeletedDraftBranch(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, false)

	mainSHA := seedMainBranch(t, ctx, r, repository, canvas.Name)

	draftBranch := models.NewDraftBranchName()
	require.NoError(t, r.GitProvider.CreateBranch(ctx, repository.RepoID, draftBranch, models.CanvasGitBranchMain))
	draftHead, err := r.GitProvider.Head(ctx, repository.RepoID, draftBranch)
	require.NoError(t, err)

	m := newBranchMaterializer(r)
	require.NoError(t, m.MaterializeBranch(ctx, canvas.ID, draftBranch, draftHead, &r.User))
	_, err = models.FindDraftVersionByBranch(canvas.ID, draftBranch)
	require.NoError(t, err)

	// Deleting the branch from git leaves an orphaned projection. It is cleaned
	// up opportunistically by the deletion sweep that runs at the start of the
	// next materialization of any branch.
	require.NoError(t, r.GitProvider.DeleteBranch(ctx, repository.RepoID, draftBranch))
	require.NoError(t, m.MaterializeBranch(ctx, canvas.ID, models.CanvasGitBranchMain, mainSHA, nil))

	_, err = models.FindDraftVersionByBranch(canvas.ID, draftBranch)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestBranchMaterializer_ReconcileBranchDeletion(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, false)

	seedMainBranch(t, ctx, r, repository, canvas.Name)

	draftBranch := models.NewDraftBranchName()
	require.NoError(t, r.GitProvider.CreateBranch(ctx, repository.RepoID, draftBranch, models.CanvasGitBranchMain))
	draftHead, err := r.GitProvider.Head(ctx, repository.RepoID, draftBranch)
	require.NoError(t, err)

	m := newBranchMaterializer(r)
	require.NoError(t, m.MaterializeBranch(ctx, canvas.ID, draftBranch, draftHead, &r.User))
	_, err = models.FindDraftVersionByBranch(canvas.ID, draftBranch)
	require.NoError(t, err)

	// The targeted entry point drops the projection of one deleted branch.
	require.NoError(t, r.GitProvider.DeleteBranch(ctx, repository.RepoID, draftBranch))
	require.NoError(t, m.ReconcileBranchDeletion(ctx, canvas.ID, draftBranch))

	_, err = models.FindDraftVersionByBranch(canvas.ID, draftBranch)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
