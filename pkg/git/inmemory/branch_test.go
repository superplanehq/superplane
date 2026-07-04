package inmemory

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/git/provider"
)

func TestProviderBranchOperations(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()
	repoID := p.GetRepositoryID(provider.RepositoryOptions{
		OrganizationID: uuid.New(),
		CanvasID:       uuid.New(),
	})

	_, err := p.CreateRepository(ctx, repoID)
	require.NoError(t, err)

	mainHead, err := p.Head(ctx, repoID, "main")
	require.NoError(t, err)

	draftBranch := "drafts/" + uuid.New().String()
	require.NoError(t, p.CreateBranch(ctx, repoID, draftBranch, "main"))

	branches, err := p.ListBranches(ctx, repoID, "drafts/")
	require.NoError(t, err)
	assert.Contains(t, branches, draftBranch)

	draftHead, err := p.Head(ctx, repoID, draftBranch)
	require.NoError(t, err)
	assert.Equal(t, mainHead, draftHead)

	commitSHA, err := p.Commit(ctx, repoID, provider.CommitOptions{
		Branch:          draftBranch,
		BaseBranch:      draftBranch,
		ExpectedHeadSHA: draftHead,
		Message:         "draft change",
		Author: provider.CommitAuthor{
			Name:  "Test",
			Email: "test@example.com",
		},
		Operations: []provider.FileOperation{
			{Path: "notes.txt", Content: bytes.NewReader([]byte("draft")), SizeBytes: 5},
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, commitSHA)

	mergedSHA, err := p.MergeBranch(ctx, repoID, draftBranch, "main", "merge draft", provider.CommitAuthor{
		Name:  "Test",
		Email: "test@example.com",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, mergedSHA)

	files, err := p.ListFiles(ctx, repoID, "main")
	require.NoError(t, err)
	assert.Contains(t, files, "notes.txt")

	require.NoError(t, p.DeleteBranch(ctx, repoID, draftBranch))
	branches, err = p.ListBranches(ctx, repoID, "drafts/")
	require.NoError(t, err)
	assert.NotContains(t, branches, draftBranch)
}
