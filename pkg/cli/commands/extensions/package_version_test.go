package extensions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePackageDestinationDefaultsToDist(t *testing.T) {
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	tempDir := t.TempDir()
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	path, err := resolvePackageDestination("")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(".", "dist"), path)
}

func TestResolvePackageDestinationAcceptsDirectory(t *testing.T) {
	path, err := resolvePackageDestination(t.TempDir())
	require.NoError(t, err)
	require.DirExists(t, path)
}

func TestResolvePackageDestinationRejectsFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "artifact")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o600))

	_, err := resolvePackageDestination(filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be a directory")
}

func TestFormatManifestJSON(t *testing.T) {
	payload, err := formatManifestJSON([]byte(`{"kind":"extension","metadata":{"id":"examples.github"}}`))
	require.NoError(t, err)
	require.Contains(t, string(payload), "\"kind\": \"extension\"")
	require.Contains(t, string(payload), "\n")
}
