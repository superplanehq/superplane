package installation

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tarEntry struct {
	name    string
	content []byte
	dir     bool
}

func buildTarball(t *testing.T, entries []tarEntry) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for _, entry := range entries {
		header := &tar.Header{
			Name:    entry.name,
			Mode:    0o644,
			Size:    int64(len(entry.content)),
			ModTime: time.Now(),
		}
		if entry.dir {
			header.Typeflag = tar.TypeDir
			header.Size = 0
		} else {
			header.Typeflag = tar.TypeReg
		}

		require.NoError(t, tw.WriteHeader(header))
		if !entry.dir {
			_, err := tw.Write(entry.content)
			require.NoError(t, err)
		}
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func stubTarball(t *testing.T, url string, payload []byte, status int) {
	t.Helper()
	original := tarballHTTPGet
	tarballHTTPGet = func(rawURL string) (*http.Response, error) {
		if rawURL != url {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		}
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(bytes.NewReader(payload)),
		}, nil
	}
	t.Cleanup(func() { tarballHTTPGet = original })
}

func TestFetchRepositoryFilesRequiresRef(t *testing.T) {
	repo := &Repository{Owner: "acme", Name: "demo"}
	_, err := FetchRepositoryFiles(repo, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolved ref")
}

func TestFetchRepositoryFilesReturnsNilWhenMissing(t *testing.T) {
	repo := &Repository{Owner: "acme", Name: "demo"}
	stubTarball(t, tarballURL(repo, "main"), nil, http.StatusNotFound)

	files, err := FetchRepositoryFiles(repo, "main")
	require.NoError(t, err)
	assert.Nil(t, files)
}

func TestFetchRepositoryFilesExcludesSpecFilesAndParams(t *testing.T) {
	repo := &Repository{Owner: "acme", Name: "demo"}
	payload := buildTarball(t, []tarEntry{
		{name: "demo-main/", dir: true},
		{name: "demo-main/canvas.yaml", content: []byte("canvas")},
		{name: "demo-main/console.yaml", content: []byte("console")},
		{name: "demo-main/params.json", content: []byte("{}")},
		{name: "demo-main/README.md", content: []byte("# hello")},
		{name: "demo-main/scripts/", dir: true},
		{name: "demo-main/scripts/deploy.sh", content: []byte("#!/bin/sh\necho hi\n")},
		{name: "demo-main/.git/config", content: []byte("[core]\n")},
	})
	stubTarball(t, tarballURL(repo, "main"), payload, http.StatusOK)

	files, err := FetchRepositoryFiles(repo, "main")
	require.NoError(t, err)

	paths := make(map[string][]byte, len(files))
	for _, file := range files {
		paths[file.Path] = file.Content
	}

	assert.Len(t, files, 2, "expected README.md and scripts/deploy.sh only")
	assert.Equal(t, []byte("# hello"), paths["README.md"])
	assert.Equal(t, []byte("#!/bin/sh\necho hi\n"), paths["scripts/deploy.sh"])
	assert.NotContains(t, paths, "canvas.yaml")
	assert.NotContains(t, paths, "console.yaml")
	assert.NotContains(t, paths, "params.json")
	assert.NotContains(t, paths, ".git/config")
}

func TestFetchRepositoryFilesSkipsReservedPaths(t *testing.T) {
	repo := &Repository{Owner: "acme", Name: "demo"}
	payload := buildTarball(t, []tarEntry{
		{name: "demo-main/", dir: true},
		// .superplane is reserved for SuperPlane use only, install must
		// not seed it.
		{name: "demo-main/.superplane/secret", content: []byte("ignored")},
		{name: "demo-main/keep.txt", content: []byte("kept")},
	})
	stubTarball(t, tarballURL(repo, "main"), payload, http.StatusOK)

	files, err := FetchRepositoryFiles(repo, "main")
	require.NoError(t, err)

	require.Len(t, files, 1)
	assert.Equal(t, "keep.txt", files[0].Path)
}

func TestFetchRepositoryFilesEnforcesFileSizeLimit(t *testing.T) {
	repo := &Repository{Owner: "acme", Name: "demo"}
	oversize := bytes.Repeat([]byte{'x'}, maxRepositoryFileSizeBytes+1)
	payload := buildTarball(t, []tarEntry{
		{name: "demo-main/", dir: true},
		{name: "demo-main/big.bin", content: oversize},
	})
	stubTarball(t, tarballURL(repo, "main"), payload, http.StatusOK)

	_, err := FetchRepositoryFiles(repo, "main")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum size")
}
