package extensions

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateBundleArchiveFromDirectory(t *testing.T) {
	root := t.TempDir()
	distDir := filepath.Join(root, "dist")
	require.NoError(t, os.MkdirAll(distDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "bundle.js"), []byte("console.log('hi');"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "manifest.json"), []byte(`{"kind":"extension"}`), 0o600))

	archive, err := createBundleArchiveFromDirectory(distDir)
	require.NoError(t, err)

	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	require.NoError(t, err)
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	files := make(map[string]string)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		contents, err := io.ReadAll(tarReader)
		require.NoError(t, err)
		files[header.Name] = string(contents)
	}

	require.Equal(t, "console.log('hi');", files["dist/bundle.js"])
	require.Equal(t, `{"kind":"extension"}`, files["dist/manifest.json"])
}
