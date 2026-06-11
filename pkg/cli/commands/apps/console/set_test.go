package console

import (
	"bytes"
	"encoding/base64"
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

// describeCanvasCMEnabledResponse returns a canvas whose spec advertises
// change management enabled. Used by tests that exercise the auto change
// request flow in `console set`.
func describeCanvasCMEnabledResponse(t *testing.T, w http.ResponseWriter, _ *http.Request) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testCanvasID + `","name":"` + testCanvasName + `"},"spec":{"changeManagement":{"enabled":true}}}}`))
}

func repositoryCommitsPath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/repository/commits"
}

func expectCommitConsoleYAML(expectedVersionID string) requestExpectation {
	return requestExpectation{
		method: http.MethodPost,
		path:   repositoryCommitsPath(testCanvasID),
		handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var parsed map[string]any
			require.NoError(t, json.Unmarshal(body, &parsed))
			require.Equal(t, expectedVersionID, parsed["versionId"])
			operations, ok := parsed["operations"].([]any)
			require.True(t, ok)
			require.NotEmpty(t, operations)
			first, ok := operations[0].(map[string]any)
			require.True(t, ok)
			encoded, ok := first["content"].(string)
			require.True(t, ok)
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			require.NoError(t, err)
			require.Contains(t, string(decoded), "notes")

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		},
	}
}

func expectFetchConsoleYAML(versionID string) requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   repositoryConsoleFilePath(testCanvasID),
		handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "console.yaml", r.URL.Query().Get("path"))
			require.Equal(t, versionID, r.URL.Query().Get("version_id"))
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(sampleConsoleYAML))
		},
	}
}

func expectCommitAndFetchConsoleYAML(versionID string) []requestExpectation {
	return []requestExpectation{
		expectCommitConsoleYAML(versionID),
		expectFetchConsoleYAML(versionID),
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
		expectListUserDraftBranch(testCanvasID, "draft-1"),
		expectCommitConsoleYAML("draft-1"),
		expectFetchConsoleYAML("draft-1"),
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
		expectListUserDraftBranch(testCanvasID, "draft-1"),
		expectCommitConsoleYAML("draft-1"),
		expectFetchConsoleYAML("draft-1"),
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
		expectListUserDraftBranch(testCanvasID, "draft-1"),
		expectCommitConsoleYAML("draft-1"),
		expectFetchConsoleYAML("draft-1"),
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
		expectListDraftBranchesEmpty(testCanvasID),
		expectCreateDraftBranch(testCanvasID, "draft-1"),
		expectCommitConsoleYAML("draft-1"),
		expectFetchConsoleYAML("draft-1"),
	)

	ctx, _ := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	require.NoError(t, (&setCommand{file: strPtr(path)}).Execute(ctx))
	server.AssertCalls(t, []string{
		http.MethodGet + " " + testDescribeCanvas,
		http.MethodGet + " " + draftVersionsPath(testCanvasID),
		http.MethodPost + " " + draftVersionsPath(testCanvasID),
		http.MethodPost + " " + repositoryCommitsPath(testCanvasID),
		http.MethodGet + " " + repositoryConsoleFilePath(testCanvasID),
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
		expectListUserDraftBranch(testCanvasID, "draft-1"),
		expectCommitConsoleYAML("draft-1"),
		expectFetchConsoleYAML("draft-1"),
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
		expectListUserDraftBranch(testCanvasID, "draft-1"),
		expectCommitConsoleYAML("draft-1"),
		expectFetchConsoleYAML("draft-1"),
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

	require.NoError(t, (&setCommand{file: strPtr(path)}).Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "Change request: cr-42")
}

// TestSetSkipsChangeRequestWithDraftID ensures `--draft-id` opts out of
// the auto change-request creation even when change management is on.
func TestSetSkipsChangeRequestWithDraftID(t *testing.T) {
	path := writeSampleConsoleYAML(t)

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   testDescribeCanvas,
			handle: describeCanvasCMEnabledResponse,
		},
		expectMe(),
		expectValidateDraftVersion(testCanvasID, "draft-1"),
		expectCommitConsoleYAML("draft-1"),
		expectFetchConsoleYAML("draft-1"),
	)

	ctx, stdout := newConsoleCommandContext(t, server.server, "text", nil)
	ctx.Args = []string{testCanvasID}

	require.NoError(t, (&setCommand{file: strPtr(path), draftID: strPtr("draft-1")}).Execute(ctx))
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
