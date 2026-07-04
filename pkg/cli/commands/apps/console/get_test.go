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
)

func repositoryConsoleFilePath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/repository/file"
}

const sampleConsoleYAMLBody = "apiVersion: v1\nkind: Console\nmetadata:\n  canvasId: " + testCanvasID + "\n  name: " + testCanvasName + "\nspec:\n  panels:\n    - id: notes\n      type: markdown\n      content:\n        body: Hello\n  layout:\n    - i: notes\n      x: 0\n      y: 0\n      w: 4\n      h: 2\n"

func describeCanvasResponse(t *testing.T, w http.ResponseWriter, _ *http.Request) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testCanvasID + `","name":"` + testCanvasName + `"},"spec":{"nodes":[],"edges":[]}}}`))
}

func liveConsoleResponse(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	require.Equal(t, "console.yaml", r.URL.Query().Get("path"))
	require.Empty(t, r.URL.Query().Get("version_id"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(sampleConsoleYAMLBody))
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
			path:   repositoryConsoleFilePath(testCanvasID),
			handle: liveConsoleResponse,
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "yaml", nil)
	ctx.Args = []string{testCanvasID}

	err := (&getCommand{}).Execute(ctx)
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
		http.MethodGet + " " + repositoryConsoleFilePath(testCanvasID),
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
			path:   repositoryConsoleFilePath(testCanvasID),
			handle: liveConsoleResponse,
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}
	require.NoError(t, (&getCommand{}).Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, "App: "+testCanvasName)
	require.Contains(t, out, "App ID: "+testCanvasID)
	require.Contains(t, out, "Panels: 1")
	require.Contains(t, out, "Layout items: 1")
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
			path:   repositoryConsoleFilePath(testCanvasID),
			handle: liveConsoleResponse,
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Config = &fakeConfig{activeApp: testCanvasID}
	ctx.Args = []string{}

	require.NoError(t, (&getCommand{}).Execute(ctx))
	require.Contains(t, stdout.String(), "App ID: "+testCanvasID)
}

// TestGetErrorsWhenNoCanvasAndNoActive surfaces a helpful error if the
// user invokes `console get` without an argument and without an active
// canvas configured.
func TestGetErrorsWhenNoCanvasAndNoActive(t *testing.T) {
	server := newAPITestServer(t)

	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Config = &fakeConfig{}
	ctx.Args = []string{}

	err := (&getCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "app-name-or-id")
}
