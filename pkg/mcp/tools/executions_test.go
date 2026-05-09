package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

const testListExecutionsResponse = `{
	"executions": [
		{
			"id": "exec-001",
			"canvasId": "canvas-001",
			"nodeId": "node-001",
			"state": "STATE_FINISHED",
			"result": "RESULT_PASSED",
			"createdAt": "2025-01-15T10:00:00Z",
			"updatedAt": "2025-01-15T10:01:00Z"
		},
		{
			"id": "exec-002",
			"canvasId": "canvas-001",
			"nodeId": "node-001",
			"state": "STATE_STARTED",
			"createdAt": "2025-01-15T11:00:00Z"
		}
	]
}`

const testDescribeExecutionResponse = `{
	"executions": [
		{
			"id": "child-001",
			"canvasId": "canvas-001",
			"nodeId": "node-002",
			"state": "STATE_FINISHED",
			"result": "RESULT_PASSED",
			"createdAt": "2025-01-15T10:00:00Z"
		}
	]
}`

func TestHandleListExecutions(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/canvases/canvas-001/nodes/node-001/executions", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testListExecutionsResponse))
	})

	ctx := context.Background()
	result, err := handleListExecutions(ctx, apiClient, "canvas-001", "node-001")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var executions []map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &executions))
	require.Len(t, executions, 2)

	require.Equal(t, "exec-001", executions[0]["id"])
	require.Equal(t, "STATE_FINISHED", executions[0]["state"])
	require.Equal(t, "RESULT_PASSED", executions[0]["result"])

	require.Equal(t, "exec-002", executions[1]["id"])
	require.Equal(t, "STATE_STARTED", executions[1]["state"])
}

func TestHandleDescribeExecution(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/canvases/canvas-001/executions/exec-001/children", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testDescribeExecutionResponse))
	})

	ctx := context.Background()
	result, err := handleDescribeExecution(ctx, apiClient, "canvas-001", "exec-001")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var response map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &response))
	require.Equal(t, "exec-001", response["execution_id"])
	require.Equal(t, "canvas-001", response["canvas_id"])

	children := response["child_executions"].([]any)
	require.Len(t, children, 1)
	child := children[0].(map[string]any)
	require.Equal(t, "child-001", child["id"])
	require.Equal(t, "node-002", child["node_id"])
}
