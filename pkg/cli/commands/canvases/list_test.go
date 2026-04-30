package canvases

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const canvasesListResponse = `{
	"canvases": [
		{
			"metadata": {
				"id": "canvas-001",
				"name": "my-canvas",
				"createdAt": "2025-01-15T10:00:00Z"
			},
			"spec": {
				"nodes": [{"id": "node-1"}],
				"edges": [{"id": "edge-1"}]
			}
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

func TestListCommandReturnsSummaryJSON(t *testing.T) {
	server := newCanvasListServer(t)
	ctx, stdout := newCreateCommandContextForTest(t, server, "json")
	full := false
	cmd := &listCommand{full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result []map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 1)
	require.Equal(t, "canvas-001", result[0]["id"])
	require.Equal(t, "my-canvas", result[0]["name"])
	require.Equal(t, "2025-01-15T10:00:00Z", result[0]["createdAt"])

	// Summary should not contain spec fields
	raw := stdout.String()
	require.NotContains(t, raw, "node-1")
	require.NotContains(t, raw, "edge-1")
}

func TestListCommandReturnsFullJSON(t *testing.T) {
	server := newCanvasListServer(t)
	ctx, stdout := newCreateCommandContextForTest(t, server, "json")
	full := true
	cmd := &listCommand{full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "node-1")
	require.Contains(t, raw, "spec")
	require.Contains(t, raw, "my-canvas")
}

func TestListCommandReturnsSummaryYAML(t *testing.T) {
	server := newCanvasListServer(t)
	ctx, stdout := newCreateCommandContextForTest(t, server, "yaml")
	full := false
	cmd := &listCommand{full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "id: canvas-001")
	require.Contains(t, raw, "name: my-canvas")
	require.NotContains(t, raw, "node-1")
}

func TestListCommandReturnsTextOutput(t *testing.T) {
	server := newCanvasListServer(t)
	ctx, stdout := newCreateCommandContextForTest(t, server, "text")
	full := false
	cmd := &listCommand{full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "ID")
	require.Contains(t, raw, "NAME")
	require.Contains(t, raw, "canvas-001")
	require.Contains(t, raw, "my-canvas")
}
