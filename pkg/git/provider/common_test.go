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

func TestNormalizePathRejectsGitDirectoryInAnyCase(t *testing.T) {
	for _, path := range []string{
		".git/config",
		".GIT/config",
		".Git/config",
		"dir/.giT/hooks/pre-commit",
		".git",
		".GIT",
	} {
		t.Run(path, func(t *testing.T) {
			_, err := NormalizePath(path)
			require.ErrorIs(t, err, ErrInvalidPath)
		})
	}
}

func TestNormalizePathAllowsNamesThatMerelyContainGit(t *testing.T) {
	for _, path := range []string{
		".gitignore",
		".github/workflows/ci.yml",
		"dir/git/file.txt",
	} {
		t.Run(path, func(t *testing.T) {
			normalized, err := NormalizePath(path)
			require.NoError(t, err)
			require.Equal(t, path, normalized)
		})
	}
}

func TestValidateUserPathRejectsReservedPathInAnyCase(t *testing.T) {
	for _, path := range []string{
		".superplane",
		".SuperPlane",
		".SUPERPLANE",
		".superplane/config",
		".SuperPlane/config",
		"/.SUPERPLANE//nested/config",
	} {
		t.Run(path, func(t *testing.T) {
			_, err := ValidateUserPath(path)
			require.ErrorIs(t, err, ErrReservedPath)
		})
	}
}

func TestValidateUserPathAllowsNamesThatMerelyStartWithReservedPrefix(t *testing.T) {
	for _, path := range []string{
		".superplane-notes.md",
		".SuperPlaneConfig",
		"dir/.superplane/config",
	} {
		t.Run(path, func(t *testing.T) {
			normalized, err := ValidateUserPath(path)
			require.NoError(t, err)
			require.Equal(t, path, normalized)
		})
	}
}

func TestIsReservedPath(t *testing.T) {
	for path, reserved := range map[string]bool{
		".superplane":        true,
		".SuperPlane":        true,
		".superplane/config": true,
		".SUPERPLANE/config": true,
		".superplane-notes":  false,
		"dir/.superplane":    false,
		"superplane/config":  false,
		"":                   false,
	} {
		t.Run(path, func(t *testing.T) {
			require.Equal(t, reserved, IsReservedPath(path))
		})
	}
}
