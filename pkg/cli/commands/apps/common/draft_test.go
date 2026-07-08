package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

func TestRequireCommitMessageErrorsOnEmpty(t *testing.T) {
	_, err := RequireCommitMessage("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "--message is required")
}

func TestEnsureLiveVersionIDReturnsNewestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-123" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"canvas-123","versionId":"version-1"}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	versionID, err := EnsureLiveVersionID(ctx, "canvas-123")
	require.NoError(t, err)
	require.Equal(t, "version-1", versionID)
}

func TestResolveLiveVersionIDUsesExplicitVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-123/versions/version-2" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"version-2"}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	versionID, err := ResolveLiveVersionID(ctx, "canvas-123", "version-2")
	require.NoError(t, err)
	require.Equal(t, "version-2", versionID)
}

func TestGetCanvasStagingReturnsStagingState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-123/staging" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"staging":{"hasStaging":true,"stagedPaths":["canvas.yaml"]}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	staging, err := GetCanvasStaging(ctx, "canvas-123")
	require.NoError(t, err)
	require.True(t, staging.GetHasStaging())
	require.Equal(t, []string{"canvas.yaml"}, staging.GetStagedPaths())
}

func TestNormalizeRepositoryPath(t *testing.T) {
	require.Equal(t, "canvas.yaml", NormalizeRepositoryPath("/canvas.yaml"))
	require.Equal(t, "docs/readme.md", NormalizeRepositoryPath(`\docs\readme.md`))
}

func TestRepositoryPathFromLocalFile(t *testing.T) {
	require.Equal(t, "canvas.yaml", RepositoryPathFromLocalFile("/tmp/work/canvas.yaml"))
}

func TestStageRepositoryFilesRequiresFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	err := StageRepositoryFiles(ctx, "canvas-123", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one file is required")
}

func TestStageRepositoryFilesStagesContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/api/v1/canvases/canvas-123/staging" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	err := StageRepositorySpecFile(ctx, "canvas-123", CanvasYAMLRepositoryPath, []byte("apiVersion: v1"))
	require.NoError(t, err)
}

func TestCommitCanvasStagingPostsMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases/canvas-123/staging/commit" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"version-2"}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	response, err := CommitCanvasStaging(ctx, "canvas-123", "Ship it")
	require.NoError(t, err)
	version, ok := response.GetVersionOk()
	require.True(t, ok)
	metadata, ok := version.GetMetadataOk()
	require.True(t, ok)
	require.Equal(t, "version-2", metadata.GetId())
}

func TestDiscardCanvasStagingDeletesStaging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/canvases/canvas-123/staging" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	require.NoError(t, DiscardCanvasStaging(ctx, "canvas-123"))
}
