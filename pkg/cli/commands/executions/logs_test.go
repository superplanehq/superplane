package executions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	broker := newExecutionLogBroker(t)
	server := newExecutionLogsServer(t, broker.URL)
	ctx, stdout := newExecutionsCommandContext(t, server, "json")
	canvasID := "canvas-001"
	executionID := "exec-001"
	empty := ""
	limit := int64(2)

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
	require.Len(t, output[0].Records, 2)
	require.True(t, output[0].Truncated)
	require.Equal(t, "hello", output[0].Records[0].Text)
}

func TestLogsCommandResolvesLatestNodeExecution(t *testing.T) {
	broker := newExecutionLogBroker(t)
	server := newExecutionLogsServer(t, broker.URL)
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

func TestLogsCommandResolvesRunExecutions(t *testing.T) {
	broker := newExecutionLogBroker(t)
	server := newExecutionLogsServer(t, broker.URL)
	ctx, stdout := newExecutionsCommandContext(t, server, "json")
	canvasID := "canvas-001"
	empty := ""
	runID := "run-001"
	limit := int64(5)

	cmd := &LogsCommand{
		CanvasID:    &canvasID,
		ExecutionID: &empty,
		RunID:       &runID,
		NodeID:      &empty,
		Limit:       &limit,
	}

	require.NoError(t, cmd.Execute(ctx))

	var output []runnerLogOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	require.Len(t, output, 1)
	require.Equal(t, "exec-001", output[0].ExecutionID)
	require.Equal(t, "node-001", output[0].NodeID)
	require.Empty(t, output[0].Error)
}

func newExecutionLogBroker(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/tasks/task-001/live-logs", r.URL.Path)
		require.Equal(t, "Bearer token-001", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = fmt.Fprintln(w, `{"type":"line","text":"hello"}`)
		_, _ = fmt.Fprintln(w, `{"type":"cmd_end","status":"passed","duration_ms":10}`)
		_, _ = fmt.Fprintln(w, `{"type":"line","text":"after limit"}`)
	}))
	t.Cleanup(server.Close)
	return server
}

func newExecutionLogsServer(t *testing.T, brokerURL string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/canvases/canvas-001/node-executions/exec-001/runner-live-logs/session":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{
				"stream_url":%q,
				"token":"token-001",
				"expires_at":"2026-07-01T10:00:00Z"
			}`, brokerURL+"/v1/tasks/task-001/live-logs")
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
