package provider

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "normal relative path is cleaned",
			input:    `  /dir\nested//file.txt  `,
			expected: "dir/nested/file.txt",
		},
		{
			name:    "empty input is rejected",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace-only input is rejected",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "parent traversal is rejected",
			input:   "../secret.txt",
			wantErr: true,
		},
		{
			name:    "git segment is rejected",
			input:   "dir/.git/config",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := NormalizePath(tt.input)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidPath)
				require.Empty(t, normalized)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, normalized)
		})
	}
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
