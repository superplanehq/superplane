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
	testMePath         = "/api/v1/me"
	cliTestUserID      = "user-1"
)

func repositoryConsoleFilePath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/repository/file"
}

const sampleConsoleYAMLBody = "apiVersion: v1\nkind: Console\nmetadata:\n  canvasId: " + testCanvasID + "\n  name: " + testCanvasName + "\nspec:\n  panels:\n    - id: notes\n      type: markdown\n      content:\n        body: Hello\n  layout:\n    - i: notes\n      x: 0\n      y: 0\n      w: 4\n      h: 2\n"

func draftVersionsPath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/versions"
}

func expectMe() requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   testMePath,
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"` + cliTestUserID + `"}}`))
		},
	}
}

func expectListUserDraftBranch(canvasID, versionID string) requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   draftVersionsPath(canvasID),
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"` + versionID + `","owner":{"id":"` + cliTestUserID + `"}}}]}`))
		},
	}
}

func expectListDraftBranchesEmpty(canvasID string) requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   draftVersionsPath(canvasID),
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[]}`))
		},
	}
}

func expectCreateDraftBranch(canvasID, versionID string) requestExpectation {
	return requestExpectation{
		method: http.MethodPost,
		path:   draftVersionsPath(canvasID),
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"` + versionID + `"}}}`))
		},
	}
}

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
	noDraft := false
	require.NoError(t, (&getCommand{draft: &noDraft}).Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, "App: "+testCanvasName)
	require.Contains(t, out, "App ID: "+testCanvasID)
	require.Contains(t, out, "Source: live")
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
			path:   draftVersionsPath(testCanvasID),
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","owner":{"id":"user-1"}}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   repositoryConsoleFilePath(testCanvasID),
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "console.yaml", r.URL.Query().Get("path"))
				require.Equal(t, "draft-1", r.URL.Query().Get("version_id"))
				require.Equal(t, "true", r.URL.Query().Get("stage"))
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, _ = w.Write([]byte("apiVersion: v1\nkind: Console\nmetadata:\n  canvasId: " + testCanvasID + "\nspec:\n  panels: []\n  layout: []\n"))
			},
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	draft := true
	require.NoError(t, (&getCommand{draft: &draft}).Execute(ctx))
	require.Contains(t, stdout.String(), "Source: draft (staged)")
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
			path:   repositoryConsoleFilePath(testCanvasID),
			handle: liveConsoleResponse,
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Config = &fakeConfig{activeApp: testCanvasID}
	ctx.Args = []string{}

	noDraft := false
	require.NoError(t, (&getCommand{draft: &noDraft}).Execute(ctx))
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

	noDraft := false
	err := (&getCommand{draft: &noDraft}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "app-name-or-id")
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
			path:   draftVersionsPath(testCanvasID),
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[]}`))
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

func TestGetDraftIDSelectsExplicitDraftVersion(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   testMePath,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"user":{"id":"user-1"}}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions/draft-2",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-2","state":"STATE_DRAFT","owner":{"id":"user-1"}}}}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   repositoryConsoleFilePath(testCanvasID),
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "console.yaml", r.URL.Query().Get("path"))
				require.Equal(t, "draft-2", r.URL.Query().Get("version_id"))
				require.Equal(t, "true", r.URL.Query().Get("stage"))
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, _ = w.Write([]byte(sampleConsoleYAMLBody))
			},
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	draftID := "draft-2"
	require.NoError(t, (&getCommand{draftID: &draftID}).Execute(ctx))
	require.Contains(t, stdout.String(), "Version ID: draft-2")
}
