package events

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const listExecutionsResponse = `{
	"executions": [
		{
			"id": "exec-001",
			"canvasId": "canvas-001",
			"nodeId": "node-001",
			"state": "STATE_FINISHED",
			"result": "RESULT_PASSED",
			"outputs": {"data": "large-output"},
			"metadata": {"key": "value"},
			"createdAt": "2025-01-15T10:00:00Z",
			"updatedAt": "2025-01-15T10:01:00Z",
			"childExecutions": [
				{"id": "child-001", "nodeId": "node-002"}
			]
		}
	]
}`

func TestListExecutionsReturnsSummaryJSON(t *testing.T) {
	server := newListExecutionsServer(t)
	ctx, stdout := newEventsCommandContext(t, server, "json")
	canvasID := "canvas-001"
	eventID := "evt-001"
	full := false
	cmd := &ListEventExecutionsCommand{CanvasID: &canvasID, EventID: &eventID, Full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result []map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 1)
	require.Equal(t, "exec-001", result[0]["id"])
	require.Equal(t, "node-001", result[0]["nodeId"])
	require.Equal(t, "STATE_FINISHED", result[0]["state"])
	require.Equal(t, "RESULT_PASSED", result[0]["result"])
	require.Equal(t, "2025-01-15T10:00:00Z", result[0]["createdAt"])
	require.Equal(t, "2025-01-15T10:01:00Z", result[0]["updatedAt"])

	raw := stdout.String()
	require.NotContains(t, raw, "large-output")
	require.NotContains(t, raw, "child-001")
}

func TestListExecutionsReturnsFullJSON(t *testing.T) {
	server := newListExecutionsServer(t)
	ctx, stdout := newEventsCommandContext(t, server, "json")
	canvasID := "canvas-001"
	eventID := "evt-001"
	full := true
	cmd := &ListEventExecutionsCommand{CanvasID: &canvasID, EventID: &eventID, Full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "large-output")
	require.Contains(t, raw, "child-001")
	require.Contains(t, raw, "exec-001")
}

func newListExecutionsServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/canvases/canvas-001/events/evt-001/executions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(listExecutionsResponse))
	}))
	t.Cleanup(server.Close)
	return server
}
