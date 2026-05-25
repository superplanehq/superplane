package console

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportSendsParsedYAMLToAPI(t *testing.T) {
	dir := t.TempDir()
	consoleFile := filepath.Join(dir, "console.yaml")
	require.NoError(t, os.WriteFile(consoleFile, []byte(`apiVersion: v1
kind: Console
metadata:
  canvasId: canvas-123
  name: Demo
spec:
  panels:
    - id: m1
      type: markdown
      content:
        title: Notes
        body: hello
  layout:
    - i: m1
      x: 0
      y: 0
      w: 4
      h: 3
`), 0o600))

	var captured map[string]any
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodPut,
		path:   "/api/v1/canvases/canvas-123/dashboard",
		handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(body, &captured))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"dashboard": {"panels": [{"id":"m1"}], "layout": [{"i":"m1"}]}}`))
		},
	})

	ctx, _ := newCommandContext(t, server.server, "text")
	cmd := &importCommand{
		canvasID: stringPtr("canvas-123"),
		file:     stringPtr(consoleFile),
		yes:      boolPtr(true),
	}

	require.NoError(t, cmd.Execute(ctx))

	panels, _ := captured["panels"].([]any)
	require.Len(t, panels, 1)
	first, _ := panels[0].(map[string]any)
	require.Equal(t, "m1", first["id"])
	require.Equal(t, "markdown", first["type"])
}

func TestImportRejectsUnsupportedKind(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "console.yaml")
	require.NoError(t, os.WriteFile(bad, []byte("apiVersion: v1\nkind: Canvas\nspec:\n  panels: []\n  layout: []\n"), 0o600))

	server := newAPITestServer(t)
	ctx, _ := newCommandContext(t, server.server, "text")

	cmd := &importCommand{
		canvasID: stringPtr("canvas-123"),
		file:     stringPtr(bad),
		yes:      boolPtr(true),
	}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported resource kind")
}

func TestImportRejectsLayoutForUnknownPanel(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "console.yaml")
	require.NoError(t, os.WriteFile(bad, []byte(`apiVersion: v1
kind: Console
spec:
  panels:
    - id: m1
      type: markdown
  layout:
    - i: missing
      x: 0
      y: 0
      w: 1
      h: 1
`), 0o600))

	server := newAPITestServer(t)
	ctx, _ := newCommandContext(t, server.server, "text")

	cmd := &importCommand{
		canvasID: stringPtr("canvas-123"),
		file:     stringPtr(bad),
		yes:      boolPtr(true),
	}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), `layout item "missing"`)
}
