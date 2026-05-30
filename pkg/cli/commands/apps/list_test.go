package apps

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const canvasesListResponse = `{
	"canvases": [
		{
			"metadata": {
				"id": "canvas-001",
				"name": "my-canvas",
				"createdAt": "2025-01-15T10:00:00Z"
			},
			"spec": {
				"nodes": [{"id": "node-1"}],
				"edges": [{"id": "edge-1"}]
			}
		}
	]
}`

func newCanvasListServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/canvases", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(canvasesListResponse))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestListCommandReturnsSummaryJSON(t *testing.T) {
	server := newCanvasListServer(t)
	ctx, stdout := newListCommandContextForTest(t, server, "json")
	full := false
	cmd := &listCommand{full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result []map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 1)
	require.Equal(t, "canvas-001", result[0]["id"])
	require.Equal(t, "my-canvas", result[0]["name"])
	require.Equal(t, "2025-01-15T10:00:00Z", result[0]["createdAt"])

	// Summary should not contain spec fields
	raw := stdout.String()
	require.NotContains(t, raw, "node-1")
	require.NotContains(t, raw, "edge-1")
}

func TestListCommandReturnsFullJSON(t *testing.T) {
	server := newCanvasListServer(t)
	ctx, stdout := newListCommandContextForTest(t, server, "json")
	full := true
	cmd := &listCommand{full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "node-1")
	require.Contains(t, raw, "spec")
	require.Contains(t, raw, "my-canvas")
}

func TestListCommandReturnsSummaryYAML(t *testing.T) {
	server := newCanvasListServer(t)
	ctx, stdout := newListCommandContextForTest(t, server, "yaml")
	full := false
	cmd := &listCommand{full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "id: canvas-001")
	require.Contains(t, raw, "name: my-canvas")
	require.NotContains(t, raw, "node-1")
}

func TestListCommandReturnsTextOutput(t *testing.T) {
	server := newCanvasListServer(t)
	ctx, stdout := newListCommandContextForTest(t, server, "text")
	full := false
	cmd := &listCommand{full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "ID")
	require.Contains(t, raw, "NAME")
	require.Contains(t, raw, "canvas-001")
	require.Contains(t, raw, "my-canvas")
}

func newListCommandContextForTest(
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
