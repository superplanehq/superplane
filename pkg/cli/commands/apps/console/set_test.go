package console

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const sampleConsoleYAML = `apiVersion: v1
kind: Console
metadata:
  canvasId: ` + testCanvasID + `
  name: ` + testCanvasName + `
spec:
  panels:
    - id: notes
      type: markdown
      content:
        body: Hello world
  layout:
    - i: notes
      x: 0
      y: 0
      w: 4
      h: 2
`

func writeSampleConsoleYAML(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "console.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleConsoleYAML), 0o600))
	return path
}

func listVersionsWithDraft(t *testing.T, w http.ResponseWriter, _ *http.Request) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","state":"STATE_DRAFT"}}],"hasNextPage":false}`))
}

// describeCanvasCMEnabledResponse returns a canvas whose spec advertises
// change management enabled. Used by tests that exercise the auto change
// request flow in `console set`.
func describeCanvasCMEnabledResponse(t *testing.T, w http.ResponseWriter, _ *http.Request) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testCanvasID + `","name":"` + testCanvasName + `"},"spec":{"changeManagement":{"enabled":true}}}}`))
}

func handleUpdateDashboard(t *testing.T, expectedVersionID string) func(*testing.T, http.ResponseWriter, *http.Request) {
	return func(_ *testing.T, w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var parsed map[string]any
		require.NoError(t, json.Unmarshal(body, &parsed))
		require.Equal(t, expectedVersionID, parsed["versionId"])

		panels, ok := parsed["panels"].([]any)
		require.True(t, ok)
		require.Len(t, panels, 1)
		firstPanel := panels[0].(map[string]any)
		require.Equal(t, "notes", firstPanel["id"])
		require.Equal(t, "markdown", firstPanel["type"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "dashboard": {
                "canvasId": "` + testCanvasID + `",
                "versionId": "` + expectedVersionID + `",
                "panels": [{"id":"notes","type":"markdown","content":{"body":"Hello world"}}],
                "layout": [{"i":"notes","x":0,"y":0,"w":4,"h":2}]
            }
        }`))
	}
}

func TestSetFromFileFlag(t *testing.T) {
	path := writeSampleConsoleYAML(t)

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions",
			handle: listVersionsWithDraft,
		},
		requestExpectation{
			method: http.MethodPut,
			path:   testDashboardPath,
			handle: handleUpdateDashboard(t, "draft-1"),
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	require.NoError(t, (&setCommand{file: strPtr(path)}).Execute(ctx))
	require.Contains(t, stdout.String(), "Console draft updated for app "+testCanvasID)
	require.Contains(t, stdout.String(), "Draft version: draft-1")
	require.Contains(t, stdout.String(), "Panels: 1")
}

func TestSetFromPositionalFile(t *testing.T) {
	path := writeSampleConsoleYAML(t)

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions",
			handle: listVersionsWithDraft,
		},
		requestExpectation{
			method: http.MethodPut,
			path:   testDashboardPath,
			handle: handleUpdateDashboard(t, "draft-1"),
		},
	)

	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID, path}

	require.NoError(t, (&setCommand{file: strPtr("")}).Execute(ctx))
}

func TestSetFromStdin(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions",
			handle: listVersionsWithDraft,
		},
		requestExpectation{
			method: http.MethodPut,
			path:   testDashboardPath,
			handle: handleUpdateDashboard(t, "draft-1"),
		},
	)

	ctx, _ := newConsoleCommandContext(t, server.server, "text", bytes.NewBufferString(sampleConsoleYAML))
	ctx.Args = []string{testCanvasID}

	require.NoError(t, (&setCommand{file: strPtr("-")}).Execute(ctx))
}

func TestSetCreatesDraftWhenMissing(t *testing.T) {
	path := writeSampleConsoleYAML(t)

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[],"hasNextPage":false}`))
			},
		},
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1"}}}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   testDashboardPath,
			handle: handleUpdateDashboard(t, "draft-1"),
		},
	)

	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	require.NoError(t, (&setCommand{file: strPtr(path)}).Execute(ctx))
	server.AssertCalls(t, []string{
		http.MethodGet + " " + testDescribeCanvas,
		http.MethodGet + " /api/v1/canvases/" + testCanvasID + "/versions",
		http.MethodPost + " /api/v1/canvases/" + testCanvasID + "/versions",
		http.MethodPut + " " + testDashboardPath,
	})
}

// TestSetUsesActiveCanvasWhenNoArg verifies that omitting the canvas
// positional argument falls back to the active canvas from configuration.
func TestSetUsesActiveCanvasWhenNoArg(t *testing.T) {
	path := writeSampleConsoleYAML(t)

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions",
			handle: listVersionsWithDraft,
		},
		requestExpectation{
			method: http.MethodPut,
			path:   testDashboardPath,
			handle: handleUpdateDashboard(t, "draft-1"),
		},
	)

	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Config = &fakeConfig{activeApp: testCanvasID}
	ctx.Args = []string{}

	require.NoError(t, (&setCommand{file: strPtr(path)}).Execute(ctx))
}

// TestSetErrorsWhenNoCanvasAndNoActive ensures we surface a friendly
// error when neither an explicit canvas argument nor an active canvas
// can be resolved.
func TestSetErrorsWhenNoCanvasAndNoActive(t *testing.T) {
	path := writeSampleConsoleYAML(t)
	server := newAPITestServer(t)

	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Config = &fakeConfig{}
	ctx.Args = []string{}

	err := (&setCommand{file: strPtr(path)}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "app-name-or-id")
}

// TestSetAutoCreatesChangeRequestWhenCMEnabled verifies that, with
// change management enabled, `console set` opens a change request for
// the updated draft so it is visible from the UI.
func TestSetAutoCreatesChangeRequestWhenCMEnabled(t *testing.T) {
	path := writeSampleConsoleYAML(t)

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasCMEnabledResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions",
			handle: listVersionsWithDraft,
		},
		requestExpectation{
			method: http.MethodPut,
			path:   testDashboardPath,
			handle: handleUpdateDashboard(t, "draft-1"),
		},
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + testCanvasID + "/change-requests",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var parsed map[string]any
				require.NoError(t, json.Unmarshal(body, &parsed))
				require.Equal(t, "draft-1", parsed["versionId"])

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"changeRequest":{"metadata":{"id":"cr-42","status":"STATUS_OPEN"}}}`))
			},
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	require.NoError(t, (&setCommand{file: strPtr(path), draftOnly: boolPtr(false)}).Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "Change request: cr-42")
}

// TestSetSkipsChangeRequestWithDraftFlag ensures `--draft` opts out of
// the auto change-request creation even when change management is on.
func TestSetSkipsChangeRequestWithDraftFlag(t *testing.T) {
	path := writeSampleConsoleYAML(t)

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasCMEnabledResponse,
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testCanvasID + "/versions",
			handle: listVersionsWithDraft,
		},
		requestExpectation{
			method: http.MethodPut,
			path:   testDashboardPath,
			handle: handleUpdateDashboard(t, "draft-1"),
		},
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	require.NoError(t, (&setCommand{file: strPtr(path), draftOnly: boolPtr(true)}).Execute(ctx))
	out := stdout.String()
	require.NotContains(t, out, "Change request:")
	require.Contains(t, out, "superplane apps change-requests create")
}

func TestSetRejectsWrongKind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wrong.yaml")
	require.NoError(t, os.WriteFile(path, []byte("apiVersion: v1\nkind: Canvas\nmetadata: {}\nspec: {panels: [], layout: []}\n"), 0o600))

	server := newAPITestServer(t)
	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	err := (&setCommand{file: strPtr(path)}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported kind")
}

func TestSetRejectsEmptyYAML(t *testing.T) {
	server := newAPITestServer(t)
	ctx, _ := newConsoleCommandContext(t, server.server, "text", bytes.NewBufferString("\n"))
	ctx.Args = []string{testCanvasID}

	err := (&setCommand{file: strPtr("-")}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty")
}

func TestSetRejectsConflictingSources(t *testing.T) {
	path := writeSampleConsoleYAML(t)
	otherPath := writeSampleConsoleYAML(t)

	server := newAPITestServer(t)
	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID, otherPath}

	err := (&setCommand{file: strPtr(path)}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not both")
}

func TestSetRequiresYAMLSource(t *testing.T) {
	server := newAPITestServer(t)
	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	err := (&setCommand{file: strPtr("")}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no YAML source provided")
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
