package executions

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestLogsCommandReturnsExecutionLogsJSON(t *testing.T) {
	server := newExecutionLogsServer(t)
	ctx, stdout := newExecutionsCommandContext(t, server, "json")
	canvasID := "canvas-001"
	executionID := "exec-001"
	empty := ""
	limit := int64(5)

	cmd := &LogsCommand{
		CanvasID:    &canvasID,
		ExecutionID: &executionID,
		RunID:       &empty,
		NodeID:      &empty,
		Limit:       &limit,
	}

	require.NoError(t, cmd.Execute(ctx))

	var output []runnerLogOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	require.Len(t, output, 1)
	require.Equal(t, "exec-001", output[0].ExecutionID)
	require.Equal(t, "task-001", output[0].BrokerTaskID)
	require.Len(t, output[0].Records, 2)
	require.Equal(t, "hello", output[0].Records[0].Text)
}

func TestLogsCommandResolvesLatestNodeExecution(t *testing.T) {
	server := newExecutionLogsServer(t)
	ctx, stdout := newExecutionsCommandContext(t, server, "text")
	canvasID := "canvas-001"
	empty := ""
	nodeID := "node-001"
	limit := int64(5)

	cmd := &LogsCommand{
		CanvasID:    &canvasID,
		ExecutionID: &empty,
		RunID:       &empty,
		NodeID:      &nodeID,
		Limit:       &limit,
	}

	require.NoError(t, cmd.Execute(ctx))

	raw := stdout.String()
	require.Contains(t, raw, "Execution")
	require.Contains(t, raw, "exec-001")
	require.Contains(t, raw, "Node")
	require.Contains(t, raw, "node-001")
	require.Contains(t, raw, "hello")
}

func newExecutionLogsServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/canvases/canvas-001/node-executions/exec-001/runner-live-logs":
			require.Equal(t, "5", r.URL.Query().Get("limit"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"canvas_id":"canvas-001",
				"execution_id":"exec-001",
				"broker_task_id":"task-001",
				"count":2,
				"records":[
					{"type":"line","text":"hello"},
					{"type":"cmd_end","status":"passed","duration_ms":10}
				]
			}`))
		case r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/executions":
			require.Equal(t, "1", r.URL.Query().Get("limit"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"executions":[
					{"id":"exec-001","nodeId":"node-001","state":"STATE_FINISHED","result":"RESULT_PASSED"}
				]
			}`))
		case strings.HasPrefix(r.URL.Path, "/api/v1/canvases/canvas-001/runs/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"run":{
					"id":"run-001",
					"canvasId":"canvas-001",
					"executions":[
						{"id":"exec-001","nodeId":"node-001","state":"STATE_FINISHED"}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newExecutionsCommandContext(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{{URL: server.URL}}

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)

	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	}, stdout
}
