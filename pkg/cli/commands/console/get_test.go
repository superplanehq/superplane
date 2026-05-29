package console

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testCanvasID       = "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	testCanvasName     = "my-canvas"
	testDescribeCanvas = "/api/v1/canvases/" + testCanvasID
	testDashboardPath  = "/api/v1/canvases/" + testCanvasID + "/dashboard"
)

func describeCanvasResponse(t *testing.T, w http.ResponseWriter, _ *http.Request) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testCanvasID + `","name":"` + testCanvasName + `"},"spec":{"nodes":[],"edges":[]}}}`))
}

func liveDashboardResponse(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	require.Empty(t, r.URL.Query().Get("versionId"))
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{
        "dashboard": {
            "canvasId": "` + testCanvasID + `",
            "versionId": "live-version-1",
            "panels": [{"id": "notes", "type": "markdown", "content": {"body": "Hello"}}],
            "layout": [{"i": "notes", "x": 0, "y": 0, "w": 4, "h": 2}]
        }
    }`))
}

func TestGetLiveYAMLOutput(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   testDashboardPath,
			handle: liveDashboardResponse,
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "yaml", nil)
	ctx.Args = []string{testCanvasID}

	noDraft := false
	err := (&getCommand{draft: &noDraft}).Execute(ctx)
	require.NoError(t, err)

	out := stdout.String()
	require.Contains(t, out, "apiVersion: v1")
	require.Contains(t, out, "kind: Console")
	require.Contains(t, out, "canvasId: "+testCanvasID)
	require.Contains(t, out, "name: "+testCanvasName)
	require.Contains(t, out, "id: notes")
	require.Contains(t, out, "type: markdown")
	require.Contains(t, out, "body: Hello")

	server.AssertCalls(t, []string{
		http.MethodGet + " " + testDescribeCanvas,
		http.MethodGet + " " + testDashboardPath,
	})
}

func TestGetTextOutputSummary(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   testDashboardPath,
			handle: liveDashboardResponse,
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}
	noDraft := false
	require.NoError(t, (&getCommand{draft: &noDraft}).Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, "Canvas: "+testCanvasName)
	require.Contains(t, out, "Canvas ID: "+testCanvasID)
	require.Contains(t, out, "Source: live")
	require.Contains(t, out, "Version ID: live-version-1")
	require.Contains(t, out, "Panels: 1")
	require.Contains(t, out, "Layout items: 1")
}

func TestGetDraftResolvesUserDraftVersion(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/me",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"user":{"id":"user-1"}}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
                    "versions": [
                        {"metadata":{"id":"pub-1","state":"STATE_PUBLISHED","owner":{"id":"user-1"}}},
                        {"metadata":{"id":"draft-1","state":"STATE_DRAFT","owner":{"id":"user-1"}}}
                    ],
                    "hasNextPage": false
                }`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   testDashboardPath,
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "draft-1", r.URL.Query().Get("versionId"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"dashboard":{"canvasId":"` + testCanvasID + `","versionId":"draft-1","panels":[],"layout":[]}}`))
			},
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	draft := true
	require.NoError(t, (&getCommand{draft: &draft}).Execute(ctx))
	require.Contains(t, stdout.String(), "Source: draft")
	require.Contains(t, stdout.String(), "Version ID: draft-1")
}

// TestGetUsesActiveCanvasWhenNoArg confirms `console get` (no positional
// arg) falls back to the active canvas configured in the user's profile.
func TestGetUsesActiveCanvasWhenNoArg(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   testDashboardPath,
			handle: liveDashboardResponse,
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Config = &fakeConfig{activeCanvas: testCanvasID}
	ctx.Args = []string{}

	noDraft := false
	require.NoError(t, (&getCommand{draft: &noDraft}).Execute(ctx))
	require.Contains(t, stdout.String(), "Canvas ID: "+testCanvasID)
}

// TestGetErrorsWhenNoCanvasAndNoActive surfaces a helpful error if the
// user invokes `console get` without an argument and without an active
// canvas configured.
func TestGetErrorsWhenNoCanvasAndNoActive(t *testing.T) {
	server := newAPITestServer(t)

	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Config = &fakeConfig{}
	ctx.Args = []string{}

	noDraft := false
	err := (&getCommand{draft: &noDraft}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "canvas-name-or-id")
}

func TestGetDraftErrorsWhenNoDraftExists(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/me",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"user":{"id":"user-1"}}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"pub-1","state":"STATE_PUBLISHED","owner":{"id":"user-1"}}}],"hasNextPage":false}`))
			},
		},
	)

	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}
	draft := true
	err := (&getCommand{draft: &draft}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "draft version not found")
}
