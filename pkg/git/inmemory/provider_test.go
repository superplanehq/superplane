package inmemory

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/git/provider"
)

func TestProviderRepositoryLifecycle(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()
	repoID := p.GetRepositoryID(provider.RepositoryOptions{
		OrganizationID: uuid.New(),
		CanvasID:       uuid.New(),
	})

	_, err := p.CreateRepository(ctx, repoID)
	require.NoError(t, err)

	headSHA, err := p.Head(ctx, repoID, "")
	require.NoError(t, err)

	commitSHA, err := p.Commit(ctx, repoID, provider.CommitOptions{
		ExpectedHeadSHA: headSHA,
		Message:         "add docs",
		Operations: []provider.FileOperation{
			{
				Path:      "docs/guide.md",
				Content:   bytes.NewReader([]byte("guide")),
				SizeBytes: 5,
			},
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, commitSHA)

	files, err := p.ListFiles(ctx, repoID, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"README.md", "docs/guide.md"}, files)

	reader, err := p.GetFile(ctx, repoID, "docs/guide.md", "")
	require.NoError(t, err)
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, "guide", string(content))

	commitSHA, err = p.Commit(ctx, repoID, provider.CommitOptions{
		ExpectedHeadSHA: headSHA,
		Message:         "stale head",
	})
	require.ErrorIs(t, err, provider.ErrExpectedHeadMismatch)

	require.NoError(t, p.DeleteRepository(ctx, repoID))
	_, err = p.Head(ctx, repoID, "")
	require.ErrorIs(t, err, provider.ErrInvalidRepositoryID)
}
