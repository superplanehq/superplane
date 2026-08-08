package provider

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommitterOrAuthor(t *testing.T) {
	author := CommitAuthor{Name: "Octo Cat", Email: "octo@github.com"}

	// No committer set: defaults to the author so the commit is never
	// attributed to the code-storage host's "ubuntu" identity.
	got := CommitterOrAuthor(CommitOptions{Author: author})
	require.Equal(t, author, got)

	// Committer set explicitly: it is used as-is.
	committer := CommitAuthor{Name: "SuperPlane", Email: "bot@superplane.local"}
	got = CommitterOrAuthor(CommitOptions{Author: author, Committer: committer})
	require.Equal(t, committer, got)

	// A committer with only a name (no email) still counts as set.
	got = CommitterOrAuthor(CommitOptions{Author: author, Committer: CommitAuthor{Name: "Bot"}})
	require.Equal(t, "Bot", got.Name)
}

func TestValidateCommitOperationsNormalizesPaths(t *testing.T) {
	operations, err := ValidateCommitOperations([]FileOperation{
		{
			Path:      `  /dir\\nested//file.txt  `,
			Content:   strings.NewReader("content"),
			SizeBytes: 7,
		},
		{
			Path:   `\dir//old.txt`,
			Delete: true,
		},
	})

	require.NoError(t, err)
	require.Equal(t, "dir/nested/file.txt", operations[0].Path)
	require.Equal(t, "dir/old.txt", operations[1].Path)
}

func TestValidateCommitOperationsUsesNormalizedPathInErrors(t *testing.T) {
	_, err := ValidateCommitOperations([]FileOperation{
		{
			Path: `/dir//file.txt`,
		},
	})

	require.ErrorContains(t, err, `content is required for "dir/file.txt"`)
}
