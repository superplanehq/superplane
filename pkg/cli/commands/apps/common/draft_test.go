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
	require.Contains(t, err.Error(), "draft id is required")
}

func TestResolveDraftVersionIDValidatesExplicitDraftID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"user-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-123/versions/draft-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1","state":"STATE_DRAFT","owner":{"id":"user-1"}}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	versionID, err := ResolveDraftVersionID(ctx, "canvas-123", "draft-1")
	require.NoError(t, err)
	require.Equal(t, "draft-1", versionID)
}

func TestResolveDraftVersionIDRejectsNonDraftExplicitID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"user-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-123/versions/live-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"live-1","state":"STATE_PUBLISHED","owner":{"id":"user-1"}}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	_, err := ResolveDraftVersionID(ctx, "canvas-123", "live-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "is not a draft")
}

func TestResolveDraftVersionIDRejectsOtherOwnerExplicitID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"user-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-123/versions/draft-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1","state":"STATE_DRAFT","owner":{"id":"user-2"}}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	_, err := ResolveDraftVersionID(ctx, "canvas-123", "draft-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not owned by the current user")
}
