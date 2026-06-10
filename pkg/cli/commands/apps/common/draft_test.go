package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

func strPtr(value string) *string {
	return &value
}

func draftVersionsPath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/versions"
}

func TestMergeDraftOrVersionIDReturnsDraftID(t *testing.T) {
	id, err := MergeDraftOrVersionID(strPtr("draft-1"), strPtr(""))
	require.NoError(t, err)
	require.Equal(t, "draft-1", id)
}

func TestMergeDraftOrVersionIDReturnsVersionID(t *testing.T) {
	id, err := MergeDraftOrVersionID(strPtr(""), strPtr("draft-2"))
	require.NoError(t, err)
	require.Equal(t, "draft-2", id)
}

func TestMergeDraftOrVersionIDMatchesSameValue(t *testing.T) {
	id, err := MergeDraftOrVersionID(strPtr("draft-1"), strPtr("draft-1"))
	require.NoError(t, err)
	require.Equal(t, "draft-1", id)
}

func TestMergeDraftOrVersionIDErrorsOnConflict(t *testing.T) {
	_, err := MergeDraftOrVersionID(strPtr("draft-1"), strPtr("draft-2"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "--draft-id and --version-id must match")
}

func TestResolveDraftVersionIDReturnsEmptyWhenNotUsingDraft(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	versionID, err := ResolveDraftVersionID(ctx, "canvas-123", DraftResolveOptions{})
	require.NoError(t, err)
	require.Empty(t, versionID)
}

func TestResolveDraftVersionIDSelectsSingleOwnedDraft(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"user-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == draftVersionsPath("canvas-123"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","owner":{"id":"user-1"}}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	versionID, err := ResolveDraftVersionID(ctx, "canvas-123", DraftResolveOptions{UseDraft: true})
	require.NoError(t, err)
	require.Equal(t, "draft-1", versionID)
}

func TestResolveDraftVersionIDErrorsWhenMultipleOwnedDrafts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"user-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == draftVersionsPath("canvas-123"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","owner":{"id":"user-1"}}},{"metadata":{"id":"draft-2","owner":{"id":"user-1"}}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	_, err := ResolveDraftVersionID(ctx, "canvas-123", DraftResolveOptions{UseDraft: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple drafts found")
}

func TestResolveDraftVersionIDCreatesDraftWhenAllowed(t *testing.T) {
	var created bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"user-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == draftVersionsPath("canvas-123"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == draftVersionsPath("canvas-123"):
			created = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-new"}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")

	versionID, err := ResolveDraftVersionID(ctx, "canvas-123", DraftResolveOptions{UseDraft: true, AllowCreate: true})
	require.NoError(t, err)
	require.Equal(t, "draft-new", versionID)
	require.True(t, created)
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

	versionID, err := ResolveDraftVersionID(ctx, "canvas-123", DraftResolveOptions{DraftID: "draft-1", UseDraft: true})
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

	_, err := ResolveDraftVersionID(ctx, "canvas-123", DraftResolveOptions{DraftID: "live-1", UseDraft: true})
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

	_, err := ResolveDraftVersionID(ctx, "canvas-123", DraftResolveOptions{DraftID: "draft-1", UseDraft: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not owned by the current user")
}
