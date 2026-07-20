package runs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDescribeRunReturnsFullJSON(t *testing.T) {
	server := newDescribeRunServer(t)
	ctx, stdout := newRunsCommandContext(t, server, "json")
	ctx.Args = []string{"run-001"}
	canvasID := "canvas-001"
	cmd := &DescribeRunCommand{AppID: &canvasID}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, "run-001", result["id"])
	require.Equal(t, "Finished", result["state"])
	require.Equal(t, "Passed", result["result"])

	rootEvent, ok := result["rootEvent"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "evt-001", rootEvent["id"])
	rootEventData, ok := rootEvent["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "large-payload", rootEventData["key"])

	executions, ok := result["executions"].([]any)
	require.True(t, ok)
	require.Len(t, executions, 2)
	execution, ok := executions[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "exec-001", execution["id"])
	require.Equal(t, "Finished", execution["state"])
	require.Equal(t, "Passed", execution["result"])

	raw := stdout.String()
	require.Contains(t, raw, "evt-001")
	require.NotContains(t, raw, "/events/")
}

func TestDescribeRunShowsRootEventPayloadInText(t *testing.T) {
	server := newDescribeRunServer(t)
	ctx, stdout := newRunsCommandContext(t, server, "text")
	ctx.Args = []string{"run-001"}
	canvasID := "canvas-001"
	cmd := &DescribeRunCommand{AppID: &canvasID}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "run-001")
	require.Contains(t, raw, "Custom Name")
	require.Contains(t, raw, "Deploy production")
	require.Contains(t, raw, "node-001")
	require.Contains(t, raw, "Finished")
	require.Contains(t, raw, "Passed")
	require.Contains(t, raw, "Duration")
	require.Contains(t, raw, "Event:")
	require.Contains(t, raw, "large-payload")
	require.NotContains(t, raw, "Payload")
	require.NotContains(t, raw, "Canvas ID")
	require.NotContains(t, raw, "Version ID")
	require.NotContains(t, raw, "Updated:")
	require.NotContains(t, raw, "MESSAGE")
	require.Contains(t, raw, " ago")
	require.Contains(t, raw, "exec-001")
}

func newDescribeRunServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.True(t, strings.HasPrefix(r.URL.Path, "/api/v1/canvases/canvas-001/runs/"))
		runID := strings.TrimPrefix(r.URL.Path, "/api/v1/canvases/canvas-001/runs/")
		require.Equal(t, "run-001", runID)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(describeRunResponse))
	}))
	t.Cleanup(server.Close)
	return server
}

const describeRunResponse = `{
	"run": {
		"id": "run-001",
		"canvasId": "canvas-001",
		"state": "STATE_FINISHED",
		"result": "RESULT_PASSED",
		"createdAt": "2025-01-15T10:00:00Z",
		"updatedAt": "2025-01-15T10:05:00Z",
		"finishedAt": "2025-01-15T10:05:00Z",
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
}`
