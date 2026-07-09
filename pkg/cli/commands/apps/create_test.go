package apps

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

func TestCreateCommandRequiresName(t *testing.T) {
	ctx, _ := cli.NewCommandContext(t, nil, "text")
	name := ""
	cmd := &createCommand{name: &name}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--name is required")
}

func TestCreateCommandPrintsCanvasOnSuccess(t *testing.T) {
	server := newCreateOnlyServer(t, "abc-123", "my-canvas", "org-uuid")
	ctx, stdout := cli.NewCommandContext(t, server, "text")
	name := "my-canvas"
	cmd := &createCommand{name: &name}

	err := cmd.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `App "my-canvas" created (ID: abc-123)`)
	require.NotContains(t, stdout.String(), "Committed version:")
}

func TestCreateCommandPrintsURLFromResponseOrgID(t *testing.T) {
	server := newCreateOnlyServer(t, "abc-123", "my-canvas", "org-uuid")
	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{
		URL: "https://app.superplane.com",
	})
	name := "my-canvas"
	cmd := &createCommand{name: &name}

	err := cmd.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "App URL: https://app.superplane.com/org-uuid/apps/abc-123")
}

func TestCreateCommandFailsOnEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	name := "my-canvas"
	cmd := &createCommand{name: &name}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "server returned an empty response")
}

func TestCreateCommandWithFilesStagesAndCommits(t *testing.T) {
	var stagedBody string
	var commitBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases":
			body, _ := io.ReadAll(r.Body)
			require.Contains(t, string(body), `"name":"My App"`)
			_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"canvas-1","name":"My App","organizationId":"org-1"}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/canvases/canvas-1/staging":
			body, _ := io.ReadAll(r.Body)
			stagedBody = string(body)
			_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":true,"stagedPaths":["canvas.yaml","README.md"]}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases/canvas-1/staging/commit":
			body, _ := io.ReadAll(r.Body)
			commitBody = string(body)
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"version-1","canvasId":"canvas-1"}}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	canvasPath := filepath.Join(dir, "canvas.yaml")
	readmePath := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(canvasPath, []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: Source Name
spec:
  nodes: []
  edges: []
`), 0o600))
	require.NoError(t, os.WriteFile(readmePath, []byte("# My App\n"), 0o600))

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	name := "My App"
	files := []string{canvasPath, readmePath}
	cmd := &createCommand{name: &name, files: &files}

	err := cmd.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `App "My App" created (ID: canvas-1)`)
	require.Contains(t, stdout.String(), "Committed version: version-1")
	require.Contains(t, stagedBody, "canvas.yaml")
	require.Contains(t, stagedBody, "README.md")
	require.Contains(t, decodeStagingPayload(t, stagedBody), "id: canvas-1")
	require.Contains(t, commitBody, `Create \"My App\"`)
}

func decodeStagingPayload(t *testing.T, body string) string {
	t.Helper()
	var payload struct {
		Operations []struct {
			Content string `json:"content"`
			Path    string `json:"path"`
		} `json:"operations"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &payload))
	var decoded strings.Builder
	for _, op := range payload.Operations {
		if op.Path == "canvas.yaml" {
			raw, err := base64.StdEncoding.DecodeString(op.Content)
			require.NoError(t, err)
			decoded.Write(raw)
		}
	}
	return decoded.String()
}

func TestCreateCommandUsesCustomCommitMessage(t *testing.T) {
	var commitBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases":
			_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"canvas-1","name":"My App","organizationId":"org-1"}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/canvases/canvas-1/staging":
			_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":true,"stagedPaths":["canvas.yaml"]}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases/canvas-1/staging/commit":
			body, _ := io.ReadAll(r.Body)
			commitBody = string(body)
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"version-1","canvasId":"canvas-1"}}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	canvasPath := filepath.Join(dir, "canvas.yaml")
	require.NoError(t, os.WriteFile(canvasPath, []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: My App
spec:
  nodes: []
  edges: []
`), 0o600))

	ctx, _ := cli.NewCommandContext(t, server, "text")
	name := "My App"
	files := []string{canvasPath}
	message := "bootstrap app"
	cmd := &createCommand{name: &name, files: &files, message: &message}

	err := cmd.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, commitBody, "bootstrap app")
}

func newCreateOnlyServer(t *testing.T, id, name, orgID string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/canvases", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, name, payload["name"])
		require.NotContains(t, strings.ToLower(string(body)), `"canvas"`)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + id + `","name":"` + name + `","organizationId":"` + orgID + `"}}}`))
	}))
	t.Cleanup(server.Close)
	return server
}
