package canvases

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/git/inmemory"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func TestCommitWorkflowVersionToGitPersistsCanvasAndExtraFiles(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	created, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvas.ID.String(), "")
	require.NoError(t, err)

	version, err := models.FindCanvasVersion(canvas.ID, uuid.MustParse(created.GetVersion().GetMetadata().GetId()))
	require.NoError(t, err)

	commitSHA, err := commitWorkflowVersionToGit(ctx, r.GitProvider, commitWorkflowVersionInput{
		Canvas:         canvas,
		Version:        version,
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User,
		Message:        "Add workflow files",
		BranchName:     models.CanvasGitBranchMain,
		ExtraGitOps: []*pb.CanvasRepositoryFileOperation{
			{Path: "notes.txt", Content: []byte("hello")},
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, commitSHA)

	files, err := r.GitProvider.ListFiles(ctx, repository.RepoID, models.CanvasGitBranchMain)
	require.NoError(t, err)
	assert.Contains(t, files, CanvasYAMLRepositoryPath)
	assert.Contains(t, files, ConsoleYAMLRepositoryPath)
	assert.Contains(t, files, "notes.txt")

	reader, err := r.GitProvider.GetFile(ctx, repository.RepoID, "notes.txt", models.CanvasGitBranchMain)
	require.NoError(t, err)
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, "hello", string(content))
}

func TestMergeWorkflowBranchInGit(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	created, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvas.ID.String(), "feat/noops")
	require.NoError(t, err)

	version, err := models.FindCanvasVersion(canvas.ID, uuid.MustParse(created.GetVersion().GetMetadata().GetId()))
	require.NoError(t, err)

	_, err = commitWorkflowVersionToGit(ctx, r.GitProvider, commitWorkflowVersionInput{
		Canvas:           canvas,
		Version:          version,
		OrganizationID:   r.Organization.ID.String(),
		UserID:           r.User,
		Message:          "Feature work",
		BranchName:       "feat/noops",
		ParentBranchName: models.CanvasGitBranchMain,
	})
	require.NoError(t, err)

	mergeSHA, err := mergeWorkflowBranchInGit(
		ctx,
		r.GitProvider,
		canvas,
		r.Organization.ID.String(),
		r.User,
		"feat/noops",
		models.CanvasGitBranchMain,
		"Merge feat/noops",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, mergeSHA)

	headSHA, err := r.GitProvider.Head(ctx, repository.RepoID, models.CanvasGitBranchMain)
	require.NoError(t, err)
	assert.Equal(t, mergeSHA, headSHA)
}
