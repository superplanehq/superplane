package canvases

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestCreateCommandPrintsCanvasOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/canvases", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"abc-123","name":"my-canvas"}}}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newCreateCommandContextForTest(t, server, "text")
	ctx.Args = []string{"my-canvas"}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `Canvas "my-canvas" created (ID: abc-123)`)
}

func TestCreateCommandReturnsJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"abc-123","name":"my-canvas"}}}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newCreateCommandContextForTest(t, server, "json")
	ctx.Args = []string{"my-canvas"}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"id": "abc-123"`)
	require.Contains(t, stdout.String(), `"name": "my-canvas"`)
}

func TestCreateCommandReturnsYAMLOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"abc-123","name":"my-canvas"}}}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newCreateCommandContextForTest(t, server, "yaml")
	ctx.Args = []string{"my-canvas"}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
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

	ctx, _ := newCreateCommandContextForTest(t, server, "text")
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

	ctx, _ := newCreateCommandContextForTest(t, server, "text")
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

	ctx, _ := newCreateCommandContextForTest(t, server, "text")
	ctx.Args = []string{"my-canvas"}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
	require.Error(t, err)
}

func TestCreateFromFilePrintsCanvasOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/canvases", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"file-id-456","name":"from-file-canvas"}}}`))
	}))
	t.Cleanup(server.Close)

	filePath := writeTestCanvasFile(t, "from-file-canvas")
	file := filePath
	ctx, stdout := newCreateCommandContextForTest(t, server, "text")
	cmd := &createCommand{file: &file}

	err := cmd.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `Canvas "from-file-canvas" created (ID: file-id-456)`)
}

func TestCreateFromFileFailsOnEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	filePath := writeTestCanvasFile(t, "from-file-canvas")
	file := filePath
	ctx, _ := newCreateCommandContextForTest(t, server, "text")
	cmd := &createCommand{file: &file}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "server returned an empty response")
}

func TestCreateFromFileReturnsJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"file-id-456","name":"from-file-canvas"}}}`))
	}))
	t.Cleanup(server.Close)

	filePath := writeTestCanvasFile(t, "from-file-canvas")
	file := filePath
	ctx, stdout := newCreateCommandContextForTest(t, server, "json")
	cmd := &createCommand{file: &file}

	err := cmd.Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"id": "file-id-456"`)
	require.Contains(t, stdout.String(), `"name": "from-file-canvas"`)
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

	ctx, _ := newCreateCommandContextWithRedirectPolicyForTest(t, redirector, "text")
	ctx.Args = []string{"my-canvas"}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to follow redirect that changes method")
}

func TestCreateCommandFailsWithoutArgs(t *testing.T) {
	ctx, _ := newCreateCommandContextForTest(t, nil, "text")
	ctx.Args = []string{}
	cmd := &createCommand{}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "either --file or <canvas-name> is required")
}

func writeTestCanvasFile(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	filePath := dir + "/canvas.yaml"
	content := []byte("apiVersion: v1\nkind: Canvas\nmetadata:\n  name: " + name + "\nspec:\n  nodes: []\n  edges: []\n")
	require.NoError(t, os.WriteFile(filePath, content, 0644))
	return filePath
}

func newCreateCommandContextWithRedirectPolicyForTest(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)

	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{
		{URL: server.URL},
	}
	config.HTTPClient = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 && req.Method != via[0].Method {
				return fmt.Errorf(
					"refusing to follow redirect that changes method from %s to %s",
					via[0].Method, req.Method,
				)
			}
			return nil
		},
	}

	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	}, stdout
}

func newCreateCommandContextForTest(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)

	commandCtx := core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		Renderer: renderer,
	}

	if server != nil {
		config := openapi_client.NewConfiguration()
		config.Servers = openapi_client.ServerConfigurations{
			{URL: server.URL},
		}
		commandCtx.API = openapi_client.NewAPIClient(config)
	}

	return commandCtx, stdout
}
