package console

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetReportsPanelCountInTextOutput(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/canvas-123/dashboard",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"dashboard": {
					"canvasId": "canvas-123",
					"panels": [
						{"id": "p1", "type": "markdown", "content": {"title": "Notes"}},
						{"id": "p2", "type": "table", "content": {}}
					],
					"layout": [
						{"i": "p1", "x": 0, "y": 0, "w": 4, "h": 3}
					]
				}
			}`))
		},
	})

	ctx, stdout := newCommandContext(t, server.server, "text")
	cmd := &getCommand{canvasID: stringPtr("canvas-123")}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "Canvas ID: canvas-123")
	require.Contains(t, out, "Panels:    2")
	require.Contains(t, out, "Layout:    1")
	require.Contains(t, out, "p1")
	require.Contains(t, out, "Notes")
}

func TestGetUsesActiveCanvasWhenFlagOmitted(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/active-456/dashboard",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"dashboard": {"canvasId": "active-456"}}`))
		},
	})

	ctx, _ := newCommandContext(t, server.server, "text")
	ctx.Config = &fakeConfig{activeCanvas: "active-456"}

	cmd := &getCommand{canvasID: stringPtr("")}
	require.NoError(t, cmd.Execute(ctx))
	server.AssertCalls(t, []string{"GET /api/v1/canvases/active-456/dashboard"})
}

func TestGetEmitsJSONForJSONOutput(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/canvas-123/dashboard",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"dashboard": {"canvasId": "canvas-123", "panels": [{"id":"p1","type":"markdown"}]}}`))
		},
	})
	ctx, stdout := newCommandContext(t, server.server, "json")

	cmd := &getCommand{canvasID: stringPtr("canvas-123")}
	require.NoError(t, cmd.Execute(ctx))
	require.Contains(t, stdout.String(), `"canvasId": "canvas-123"`)
	require.Contains(t, stdout.String(), `"id": "p1"`)
}

func (s *apiTestServer) AssertCalls(t *testing.T, calls []string) {
	t.Helper()
	require.Equal(t, calls, s.calls)
	require.Len(t, s.expectations, 0, "unused request expectations")
}
