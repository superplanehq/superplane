package apps

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

func newCanvasCreateServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/canvases", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"abc-123","name":"my-canvas","organizationId":"org-uuid"}}}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestCreateCommandPrintsCanvasOnSuccess(t *testing.T) {
	server := newCanvasCreateServer(t)
	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{"my-canvas"}

	err := (&createCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `App "my-canvas" created (ID: abc-123)`)
	require.NotContains(t, stdout.String(), "App URL:")
}

func TestCreateCommandPrintsURLFromResponseOrgID(t *testing.T) {
	server := newCanvasCreateServer(t)
	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{
		URL: "https://app.superplane.com",
	})
	ctx.Args = []string{"my-canvas"}

	err := (&createCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `App "my-canvas" created (ID: abc-123)`)
	require.Contains(t, stdout.String(), "App URL: https://app.superplane.com/org-uuid/apps/abc-123")
}

func TestCreateCommandSkipsURLWhenResponseMissingOrgID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"abc-123","name":"my-canvas"}}}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{
		URL: "https://app.superplane.com",
	})
	ctx.Args = []string{"my-canvas"}

	err := (&createCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `App "my-canvas" created (ID: abc-123)`)
	require.NotContains(t, stdout.String(), "App URL:")
}

func TestCreateCommandReturnsJSONOutput(t *testing.T) {
	server := newCanvasCreateServer(t)
	ctx, stdout := cli.NewCommandContext(t, server, "json")
	ctx.Args = []string{"my-canvas"}

	err := (&createCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"id": "abc-123"`)
	require.Contains(t, stdout.String(), `"name": "my-canvas"`)
}

func TestCreateCommandReturnsYAMLOutput(t *testing.T) {
	server := newCanvasCreateServer(t)
	ctx, stdout := cli.NewCommandContext(t, server, "yaml")
	ctx.Args = []string{"my-canvas"}

	err := (&createCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "id: abc-123")
	require.Contains(t, stdout.String(), "name: my-canvas")
}

func TestCreateCommandFailsOnEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{"my-canvas"}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "server returned an empty response")
}

func TestCreateCommandFailsOnEmptyCanvasID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"","name":"my-canvas"}}}`))
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{"my-canvas"}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "server returned an empty response")
}

func TestCreateCommandFailsOnServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":13,"message":"internal error"}`))
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{"my-canvas"}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
	require.Error(t, err)
}

func TestCreateFromFilePrintsCanvasOnSuccess(t *testing.T) {
	server := newCanvasCreateServer(t)
	filePath := writeTestCanvasFile(t, "from-file-canvas")
	file := filePath
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	err := (&createCommand{canvasFile: &file}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "created (ID: abc-123)")
}

func TestCreateFromFileFailsOnEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	filePath := writeTestCanvasFile(t, "from-file-canvas")
	file := filePath
	ctx, _ := cli.NewCommandContext(t, server, "text")
	cmd := &createCommand{canvasFile: &file}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "server returned an empty response")
}

func TestCreateFromFileReturnsJSONOutput(t *testing.T) {
	server := newCanvasCreateServer(t)
	filePath := writeTestCanvasFile(t, "from-file-canvas")
	file := filePath
	ctx, stdout := cli.NewCommandContext(t, server, "json")

	err := (&createCommand{canvasFile: &file}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"id": "abc-123"`)
}

func TestCreateCommandFailsOnMethodChangingRedirect(t *testing.T) {
	// Simulates the original bug: http:// context 301-redirects POST→GET
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the redirect was followed as GET, this would succeed silently
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvases":[]}`))
	}))
	t.Cleanup(target.Close)

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusMovedPermanently)
	}))
	t.Cleanup(redirector.Close)

	ctx, _ := cli.NewCommandContextWithRedirectPolicy(t, redirector, "text")
	ctx.Args = []string{"my-canvas"}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to follow redirect that changes method")
}

func TestCreateCommandFailsWithoutArgs(t *testing.T) {
	ctx, _ := cli.NewCommandContext(t, nil, "text")
	ctx.Args = []string{}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "either --canvas-file or <app-name> is required")
}

func writeTestCanvasFile(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	filePath := dir + "/canvas.yaml"
	content := []byte("apiVersion: v1\nkind: Canvas\nmetadata:\n  name: " + name + "\nspec:\n  nodes: []\n  edges: []\n")
	require.NoError(t, os.WriteFile(filePath, content, 0644))
	return filePath
}
