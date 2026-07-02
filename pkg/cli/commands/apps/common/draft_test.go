package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

func TestResolveDraftVersionIDErrorsOnEmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	_, err := ResolveDraftVersionID(ctx, "canvas-123", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "version id is required")
}

func TestResolveDraftVersionIDValidatesExplicitVersionID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-123/versions/version-1" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"version-1"}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	versionID, err := ResolveDraftVersionID(ctx, "canvas-123", "version-1")
	require.NoError(t, err)
	require.Equal(t, "version-1", versionID)
}

func TestEnsureLiveVersionIDReturnsNewestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-123/versions" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"version-1"}}]}`))
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
