package provider

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
