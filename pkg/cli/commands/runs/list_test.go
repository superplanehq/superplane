package runs

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

const canvasRunsResponse = `{
	"runs": [
		{
			"id": "run-001",
			"canvasId": "canvas-001",
			"state": "STATE_FINISHED",
			"result": "RESULT_PASSED",
			"createdAt": "2025-01-15T10:00:00Z",
			"updatedAt": "2025-01-15T10:05:00Z",
			"rootEvent": {
				"id": "evt-001",
				"canvasId": "canvas-001",
				"nodeId": "node-001",
				"channel": "default",
				"customName": "Deploy production",
				"data": {"key": "large-payload"},
				"createdAt": "2025-01-15T10:00:00Z"
			},
			"executions": [
				{
					"id": "exec-001",
					"nodeId": "node-001",
					"state": "STATE_FINISHED",
					"result": "RESULT_PASSED",
					"createdAt": "2025-01-15T10:00:00Z"
				},
				{"id": "exec-002", "nodeId": "node-002"}
			]
		}
	],
	"totalCount": 1,
	"hasNextPage": false
}`

func TestListRunsReturnsSummaryJSON(t *testing.T) {
	server := newCanvasRunsServer(t)
	ctx, stdout := newRunsCommandContext(t, server, "json")
	canvasID := "canvas-001"
	cmd := &ListRunsCommand{AppID: &canvasID}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 1)
	require.Equal(t, "run-001", result[0]["id"])
	require.Equal(t, "node-001", result[0]["nodeId"])
	require.Equal(t, "Deploy production", result[0]["customName"])
	require.Equal(t, "Finished", result[0]["state"])
	require.Equal(t, "Passed", result[0]["result"])
	require.Equal(t, float64(2), result[0]["executions"])
	require.Contains(t, result[0]["createdAt"], " ago")

	raw := stdout.String()
	require.NotContains(t, raw, "large-payload")
	require.NotContains(t, raw, "exec-001")
}

func newCanvasRunsServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/canvases/canvas-001/runs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(canvasRunsResponse))
	}))
	t.Cleanup(server.Close)
	return server
}

func newRunsCommandContext(
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

	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	}, stdout
}
