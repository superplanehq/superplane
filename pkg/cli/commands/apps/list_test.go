package apps

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const canvasesListResponse = `{
	"canvases": [
		{
			"id": "canvas-001",
			"name": "my-canvas",
			"createdAt": "2025-01-15T10:00:00Z",
			"nodes": [{"id": "node-1"}],
			"edges": [{"id": "edge-1"}]
		}
	]
}`

func newCanvasListServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/canvases", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(canvasesListResponse))
	}))
	t.Cleanup(server.Close)
	return server
}
