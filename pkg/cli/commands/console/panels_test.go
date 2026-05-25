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

func TestPanelsListPrintsTableInTextOutput(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/canvas-123/dashboard",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"dashboard": {"panels": [{"id":"p1","type":"markdown","content":{"title":"T1"}}]}}`))
		},
	})

	ctx, stdout := newCommandContext(t, server.server, "text")
	cmd := &panelsListCommand{canvasID: stringPtr("canvas-123")}

	require.NoError(t, cmd.Execute(ctx))
	require.Contains(t, stdout.String(), "ID")
	require.Contains(t, stdout.String(), "p1")
	require.Contains(t, stdout.String(), "T1")
}

func TestPanelsGetReturns404WhenPanelMissing(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/canvas-123/dashboard",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"dashboard": {"panels": [{"id":"other","type":"markdown"}]}}`))
		},
	})

	ctx, _ := newCommandContext(t, server.server, "text")
	ctx.Args = []string{"missing"}
	cmd := &panelsGetCommand{canvasID: stringPtr("canvas-123")}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing")
}

func TestPanelsUpsertAppendsNewPanel(t *testing.T) {
	dir := t.TempDir()
	panelFile := filepath.Join(dir, "panel.yaml")
	require.NoError(t, os.WriteFile(panelFile, []byte(`id: new-panel
type: markdown
content:
  title: Hi
layout:
  x: 0
  y: 0
  w: 4
  h: 3
`), 0o600))

	var captured map[string]any
	server := newAPITestServer(t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123/dashboard",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"dashboard": {"panels": [{"id":"existing","type":"markdown"}], "layout": []}}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/canvas-123/dashboard",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(body, &captured))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"dashboard": {}}`))
			},
		},
	)

	ctx, _ := newCommandContext(t, server.server, "text")
	cmd := &panelsUpsertCommand{
		canvasID: stringPtr("canvas-123"),
		file:     stringPtr(panelFile),
		layout:   stringPtr(""),
	}
	require.NoError(t, cmd.Execute(ctx))

	panels, _ := captured["panels"].([]any)
	require.Len(t, panels, 2)
	last := panels[len(panels)-1].(map[string]any)
	require.Equal(t, "new-panel", last["id"])

	layout, _ := captured["layout"].([]any)
	require.Len(t, layout, 1)
	first, _ := layout[0].(map[string]any)
	require.Equal(t, "new-panel", first["i"])
}

func TestPanelsDeleteFiltersTargetPanel(t *testing.T) {
	var captured map[string]any
	server := newAPITestServer(t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123/dashboard",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"dashboard": {"panels":[{"id":"keep","type":"markdown"},{"id":"drop","type":"markdown"}], "layout":[{"i":"drop","x":0,"y":0,"w":1,"h":1}]}}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/canvas-123/dashboard",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(body, &captured))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"dashboard": {}}`))
			},
		},
	)

	ctx, _ := newCommandContext(t, server.server, "text")
	ctx.Args = []string{"drop"}
	cmd := &panelsDeleteCommand{canvasID: stringPtr("canvas-123"), yes: boolPtr(true)}

	require.NoError(t, cmd.Execute(ctx))

	panels, _ := captured["panels"].([]any)
	require.Len(t, panels, 1)
	require.Equal(t, "keep", panels[0].(map[string]any)["id"])

	layout, _ := captured["layout"].([]any)
	require.Len(t, layout, 0)
}
