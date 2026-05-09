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
			"state": "STATE_COMPLETED",
			"result": "RESULT_SUCCESS",
			"createdAt": "2025-01-15T10:00:00Z",
			"updatedAt": "2025-01-15T10:01:00Z"
		},
		{
			"id": "exec-002",
			"canvasId": "canvas-001",
			"nodeId": "node-001",
			"state": "STATE_RUNNING",
			"createdAt": "2025-01-15T11:00:00Z"
		}
	]
}`

const testDescribeExecutionResponse = `{
	"execution": {
		"id": "exec-001",
		"canvasId": "canvas-001",
		"nodeId": "node-001",
		"parentExecutionId": "exec-parent",
		"previousExecutionId": "exec-prev",
		"state": "STATE_COMPLETED",
		"result": "RESULT_SUCCESS",
		"resultReason": "RESULT_REASON_OK",
		"resultMessage": "Completed successfully",
		"outputs": {"key": "value"},
		"metadata": {"version": "1.0"},
		"configuration": {"timeout": 30},
		"createdAt": "2025-01-15T10:00:00Z",
		"updatedAt": "2025-01-15T10:01:00Z"
	}
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

	// Check first execution
	require.Equal(t, "exec-001", executions[0]["id"])
	require.Equal(t, "STATE_COMPLETED", executions[0]["state"])
	require.Equal(t, "RESULT_SUCCESS", executions[0]["result"])
	require.Contains(t, executions[0], "created_at")
	require.Contains(t, executions[0], "updated_at")

	// Check second execution
	require.Equal(t, "exec-002", executions[1]["id"])
	require.Equal(t, "STATE_RUNNING", executions[1]["state"])
	require.Contains(t, executions[1], "created_at")
	require.NotContains(t, executions[1], "result")
}

func TestHandleDescribeExecution(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/canvases/canvas-001/executions/exec-001/children", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

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

	var execution map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &execution))
	require.Equal(t, "exec-001", execution["id"])
	require.Equal(t, "canvas-001", execution["canvas_id"])
	require.Equal(t, "node-001", execution["node_id"])
	require.Equal(t, "exec-parent", execution["parent_execution_id"])
	require.Equal(t, "exec-prev", execution["previous_execution_id"])
	require.Equal(t, "STATE_COMPLETED", execution["state"])
	require.Equal(t, "RESULT_SUCCESS", execution["result"])
	require.Equal(t, "RESULT_REASON_OK", execution["result_reason"])
	require.Equal(t, "Completed successfully", execution["result_message"])
	require.Contains(t, execution, "outputs")
	require.Contains(t, execution, "metadata")
	require.Contains(t, execution, "configuration")
	require.Contains(t, execution, "created_at")
	require.Contains(t, execution, "updated_at")
}
