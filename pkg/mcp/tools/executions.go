package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// RegisterExecutionTools registers execution-related MCP tools
func RegisterExecutionTools(ctx context.Context, s *mcp.Server, apiClient *openapi_client.APIClient) error {
	// list_executions tool
	listExecutionsHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID string `json:"canvas_id"`
			NodeID   string `json:"node_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleListExecutions(ctx, apiClient, args.CanvasID, args.NodeID)
	}

	s.AddTool(&mcp.Tool{
		Name:        "list_executions",
		Description: "List recent executions for a canvas node. Returns execution ID, state, result, and timestamps.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"canvas_id":{"type":"string","description":"The ID of the canvas"},"node_id":{"type":"string","description":"The ID of the node"}},"required":["canvas_id","node_id"]}`),
	}, listExecutionsHandler)

	// describe_execution tool
	describeExecutionHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID    string `json:"canvas_id"`
			ExecutionID string `json:"execution_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleDescribeExecution(ctx, apiClient, args.CanvasID, args.ExecutionID)
	}

	s.AddTool(&mcp.Tool{
		Name:        "describe_execution",
		Description: "Get full details of a specific execution including state, result, outputs, and metadata.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"canvas_id":{"type":"string","description":"The ID of the canvas"},"execution_id":{"type":"string","description":"The ID of the execution"}},"required":["canvas_id","execution_id"]}`),
	}, describeExecutionHandler)

	return nil
}

// handleListExecutions lists executions for a node
func handleListExecutions(ctx context.Context, apiClient *openapi_client.APIClient, canvasID, nodeID string) (*mcp.CallToolResult, error) {
	response, _, err := apiClient.CanvasNodeAPI.CanvasesListNodeExecutions(ctx, canvasID, nodeID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	executions := response.GetExecutions()
	results := make([]map[string]any, 0, len(executions))

	for _, execution := range executions {
		result := map[string]any{
			"id": execution.GetId(),
		}

		if execution.HasState() {
			result["state"] = execution.GetState()
		}

		if execution.HasResult() {
			result["result"] = execution.GetResult()
		}

		if execution.HasCreatedAt() {
			result["created_at"] = execution.GetCreatedAt()
		}

		if execution.HasUpdatedAt() {
			result["updated_at"] = execution.GetUpdatedAt()
		}

		results = append(results, result)
	}

	content, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
	}, nil
}

// handleDescribeExecution describes a single execution with full details
func handleDescribeExecution(ctx context.Context, apiClient *openapi_client.APIClient, canvasID, executionID string) (*mcp.CallToolResult, error) {
	// Use list child executions API to get full execution details
	emptyBody := make(map[string]interface{})
	response, _, err := apiClient.CanvasNodeExecutionAPI.CanvasesListChildExecutions(ctx, canvasID, executionID).Body(emptyBody).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to describe execution: %w", err)
	}

	execution := response.GetExecution()
	result := map[string]any{
		"id":        execution.GetId(),
		"canvas_id": execution.GetCanvasId(),
		"node_id":   execution.GetNodeId(),
	}

	if execution.HasState() {
		result["state"] = execution.GetState()
	}

	if execution.HasResult() {
		result["result"] = execution.GetResult()
	}

	if execution.HasResultReason() {
		result["result_reason"] = execution.GetResultReason()
	}

	if execution.HasResultMessage() {
		result["result_message"] = execution.GetResultMessage()
	}

	if execution.HasCreatedAt() {
		result["created_at"] = execution.GetCreatedAt()
	}

	if execution.HasUpdatedAt() {
		result["updated_at"] = execution.GetUpdatedAt()
	}

	if execution.HasParentExecutionId() {
		result["parent_execution_id"] = execution.GetParentExecutionId()
	}

	if execution.HasPreviousExecutionId() {
		result["previous_execution_id"] = execution.GetPreviousExecutionId()
	}

	if execution.HasOutputs() {
		result["outputs"] = execution.GetOutputs()
	}

	if execution.HasMetadata() {
		result["metadata"] = execution.GetMetadata()
	}

	if execution.HasConfiguration() {
		result["configuration"] = execution.GetConfiguration()
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
	}, nil
}
