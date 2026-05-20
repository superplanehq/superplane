package canvases

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDashboardGetCommandReturnsYAML(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + canvasID + `","name":"deploy"}}}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + canvasID + "/dashboard",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"dashboard":{"canvasId":"` + canvasID + `","panels":[{"id":"p1","type":"markdown","content":{"body":"ok"}}],"layout":[{"i":"p1","x":0,"y":0,"w":4,"h":3}]}}`))
			},
		},
	)

	ctx, stdout := newCreateCommandContextForTest(t, server.server, "yaml")
	ctx.Args = []string{canvasID}

	err := (&dashboardGetCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "kind: Dashboard")
	require.Contains(t, stdout.String(), "canvasId: "+canvasID)
	require.Contains(t, stdout.String(), "name: deploy")
	require.Contains(t, stdout.String(), "type: markdown")
}

func TestDashboardUpdateCommandSendsPanelsAndLayout(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + canvasID + "/dashboard",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Len(t, body["panels"], 1)
				require.Len(t, body["layout"], 1)

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"dashboard":{"canvasId":"` + canvasID + `","panels":[{"id":"p1","type":"markdown","content":{"title":"Deploy"}}],"layout":[{"i":"p1","x":0,"y":0,"w":4,"h":3}]}}`))
			},
		},
	)

	filePath := writeTestDashboardFile(t)
	ctx, stdout := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{canvasID}

	err := (&dashboardUpdateCommand{file: &filePath}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Dashboard updated for canvas "+canvasID)
	require.Contains(t, stdout.String(), "Panels: 1")
}

func TestDashboardUpdateCommandRequiresDashboardKind(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "canvas.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("apiVersion: v1\nkind: Canvas\nmetadata: {}\n"), 0o644))

	ctx, _ := newCreateCommandContextForTest(t, nil, "text")
	ctx.Args = []string{"4e9ae08d-0363-40d2-ba2c-5f6389a418d8"}

	err := (&dashboardUpdateCommand{file: &filePath}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unsupported resource kind "Canvas"`)
}

func writeTestDashboardFile(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "dashboard.yaml")
	content := []byte(`
apiVersion: v1
kind: Dashboard
metadata:
  name: Deploy
spec:
  panels:
    - id: p1
      type: markdown
      content:
        title: Deploy
  layout:
    - i: p1
      x: 0
      y: 0
      w: 4
      h: 3
`)
	require.NoError(t, os.WriteFile(filePath, content, 0o644))
	return filePath
}
