package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCLIAssetName(t *testing.T) {
	require.Equal(t, "superplane-cli-darwin-arm64", cliAssetName("darwin", "arm64"))
	require.Equal(t, "superplane-cli-linux-amd64", cliAssetName("linux", "amd64"))
}

func TestCLIDownloadURL(t *testing.T) {
	require.Equal(
		t,
		"https://github.com/superplanehq/superplane/releases/download/v0.17.0/superplane-cli-linux-amd64",
		cliDownloadURL("v0.17.0", "linux", "amd64"),
	)
}

func TestReleaseAssetForPlatformUsesPublishedAssetURL(t *testing.T) {
	release := &releaseInfo{
		TagName: "v0.17.0",
		Assets: []releaseAsset{
			{
				Name:               "superplane-cli-linux-amd64",
				BrowserDownloadURL: "https://example.com/linux-amd64",
			},
		},
	}

	asset, err := release.assetForPlatform("linux", "amd64")
	require.NoError(t, err)
	require.Equal(t, "superplane-cli-linux-amd64", asset.Name)
	require.Equal(t, "https://example.com/linux-amd64", asset.BrowserDownloadURL)
}

func TestReleaseAssetForPlatformFallsBackToDeterministicURL(t *testing.T) {
	release := &releaseInfo{
		TagName: "v0.17.0",
		Assets: []releaseAsset{
			{
				Name: "superplane-cli-darwin-arm64",
			},
		},
	}

	asset, err := release.assetForPlatform("darwin", "arm64")
	require.NoError(t, err)
	require.Equal(
		t,
		"https://github.com/superplanehq/superplane/releases/download/v0.17.0/superplane-cli-darwin-arm64",
		asset.BrowserDownloadURL,
	)
}

func TestReleaseAssetForPlatformReturnsHelpfulErrorWhenMissing(t *testing.T) {
	release := &releaseInfo{TagName: "v0.17.0"}

	_, err := release.assetForPlatform("linux", "arm64")
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected superplane-cli-linux-arm64")
}

func TestDownloadAndReplaceBinary(t *testing.T) {
	tmpDir := t.TempDir()
	executablePath := filepath.Join(tmpDir, "superplane")
	require.NoError(t, os.WriteFile(executablePath, []byte("old-binary"), 0o755))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte("new-binary"))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	err := downloadAndReplaceBinary(context.Background(), server.URL, executablePath)
	require.NoError(t, err)

	contents, readErr := os.ReadFile(executablePath)
	require.NoError(t, readErr)
	require.Equal(t, "new-binary", string(contents))

	info, statErr := os.Stat(executablePath)
	require.NoError(t, statErr)
	require.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}

func TestDownloadAndReplaceBinaryRejectsEmptyDownload(t *testing.T) {
	tmpDir := t.TempDir()
	executablePath := filepath.Join(tmpDir, "superplane")
	require.NoError(t, os.WriteFile(executablePath, []byte("old-binary"), 0o755))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	err := downloadAndReplaceBinary(context.Background(), server.URL, executablePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "downloaded binary is empty")

	contents, readErr := os.ReadFile(executablePath)
	require.NoError(t, readErr)
	require.Equal(t, "old-binary", string(contents))
}

func TestDownloadAndReplaceBinaryReadsStreamedBodyAfterHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	executablePath := filepath.Join(tmpDir, "superplane")
	require.NoError(t, os.WriteFile(executablePath, []byte("old-binary"), 0o755))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		_, err := fmt.Fprint(w, "chunk-one-")
		require.NoError(t, err)
		flusher.Flush()

		time.Sleep(50 * time.Millisecond)

		_, err = fmt.Fprint(w, "chunk-two")
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	err := downloadAndReplaceBinary(context.Background(), server.URL, executablePath)
	require.NoError(t, err)

	contents, readErr := os.ReadFile(executablePath)
	require.NoError(t, readErr)
	require.Equal(t, "chunk-one-chunk-two", string(contents))
}
