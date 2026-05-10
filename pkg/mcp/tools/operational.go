package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// RegisterOperationalTools registers canvas operational tools (delete, pause, cancel)
func RegisterOperationalTools(ctx context.Context, s *mcp.Server, apiClient *openapi_client.APIClient) error {
	// delete_canvas tool
	deleteCanvasHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID string `json:"canvas_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleDeleteCanvas(ctx, apiClient, args.CanvasID)
	}

	s.AddTool(&mcp.Tool{
		Name:        "delete_canvas",
		Description: "Delete a canvas permanently. This action cannot be undone.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"canvas_id":{"type":"string","description":"The ID of the canvas to delete"}},"required":["canvas_id"]}`),
	}, deleteCanvasHandler)

	// pause_node tool
	pauseNodeHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID string `json:"canvas_id"`
			NodeID   string `json:"node_id"`
			Paused   bool   `json:"paused"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handlePauseNode(ctx, apiClient, args.CanvasID, args.NodeID, args.Paused)
	}

	s.AddTool(&mcp.Tool{
		Name:        "pause_node",
		Description: "Pause or unpause a canvas node. When paused, the node will not process incoming events.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"canvas_id":{"type":"string","description":"The ID of the canvas"},"node_id":{"type":"string","description":"The ID of the node to pause/unpause"},"paused":{"type":"boolean","description":"true to pause, false to unpause"}},"required":["canvas_id","node_id","paused"]}`),
	}, pauseNodeHandler)

	// cancel_execution tool
	cancelExecutionHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID    string `json:"canvas_id"`
			ExecutionID string `json:"execution_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleCancelExecution(ctx, apiClient, args.CanvasID, args.ExecutionID)
	}

	s.AddTool(&mcp.Tool{
		Name:        "cancel_execution",
		Description: "Cancel a running execution.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"canvas_id":{"type":"string","description":"The ID of the canvas"},"execution_id":{"type":"string","description":"The ID of the execution to cancel"}},"required":["canvas_id","execution_id"]}`),
	}, cancelExecutionHandler)
	return nil
}

// RegisterDiscoveryTools registers component discovery tools (triggers, actions)
func RegisterDiscoveryTools(ctx context.Context, s *mcp.Server, apiClient *openapi_client.APIClient) error {
	// list_triggers tool
	listTriggersHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListTriggers(ctx, apiClient)
	}

	s.AddTool(&mcp.Tool{
		Name:        "list_triggers",
		Description: "List all available trigger components. Triggers start workflow executions when events occur (e.g. webhook, schedule, GitHub push).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, listTriggersHandler)

	// list_actions tool
	listActionsHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListActions(ctx, apiClient)
	}

	s.AddTool(&mcp.Tool{
		Name:        "list_actions",
		Description: "List all available action components. Actions perform work in a workflow (e.g. run script, deploy, send notification).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, listActionsHandler)
	return nil
}

func handleDeleteCanvas(ctx context.Context, apiClient *openapi_client.APIClient, canvasID string) (*mcp.CallToolResult, error) {
	_, _, err := apiClient.CanvasAPI.CanvasesDeleteCanvas(ctx, canvasID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to delete canvas: %w", err)
	}

	result := map[string]string{
		"message":   "Canvas deleted successfully",
		"canvas_id": canvasID,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil
}

func handlePauseNode(ctx context.Context, apiClient *openapi_client.APIClient, canvasID, nodeID string, paused bool) (*mcp.CallToolResult, error) {
	body := openapi_client.CanvasesUpdateNodePauseBody{}
	body.SetPaused(paused)

	_, _, err := apiClient.CanvasNodeAPI.CanvasesUpdateNodePause(ctx, canvasID, nodeID).Body(body).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to update node pause state: %w", err)
	}

	action := "paused"
	if !paused {
		action = "unpaused"
	}

	result := map[string]string{
		"message":   fmt.Sprintf("Node %s successfully", action),
		"canvas_id": canvasID,
		"node_id":   nodeID,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil
}

func handleCancelExecution(ctx context.Context, apiClient *openapi_client.APIClient, canvasID, executionID string) (*mcp.CallToolResult, error) {
	_, _, err := apiClient.CanvasNodeExecutionAPI.CanvasesCancelExecution(ctx, canvasID, executionID).Body(map[string]interface{}{}).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to cancel execution: %w", err)
	}

	result := map[string]string{
		"message":      "Execution cancelled successfully",
		"canvas_id":    canvasID,
		"execution_id": executionID,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil
}

func handleListTriggers(ctx context.Context, apiClient *openapi_client.APIClient) (*mcp.CallToolResult, error) {
	response, _, err := apiClient.TriggerAPI.TriggersListTriggers(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list triggers: %w", err)
	}

	type triggerSummary struct {
		Name         string   `json:"name"`
		Label        string   `json:"label"`
		Description  string   `json:"description"`
		ConfigFields []string `json:"config_fields,omitempty"`
	}

	triggers := make([]triggerSummary, 0, len(response.Triggers))
	for _, t := range response.Triggers {
		fields := make([]string, 0, len(t.Configuration))
		for _, f := range t.Configuration {
			if f.Name != nil {
				fields = append(fields, *f.Name)
			}
		}
		triggers = append(triggers, triggerSummary{
			Name:         deref(t.Name),
			Label:        deref(t.Label),
			Description:  deref(t.Description),
			ConfigFields: fields,
		})
	}

	resultJSON, _ := json.MarshalIndent(triggers, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil
}

func handleListActions(ctx context.Context, apiClient *openapi_client.APIClient) (*mcp.CallToolResult, error) {
	response, _, err := apiClient.ActionAPI.ActionsListActions(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list actions: %w", err)
	}

	type actionSummary struct {
		Name           string   `json:"name"`
		Label          string   `json:"label"`
		Description    string   `json:"description"`
		ConfigFields   []string `json:"config_fields,omitempty"`
		OutputChannels []string `json:"output_channels,omitempty"`
	}

	actions := make([]actionSummary, 0, len(response.Actions))
	for _, a := range response.Actions {
		fields := make([]string, 0, len(a.Configuration))
		for _, f := range a.Configuration {
			if f.Name != nil {
				fields = append(fields, *f.Name)
			}
		}
		channels := make([]string, 0, len(a.OutputChannels))
		for _, ch := range a.OutputChannels {
			if ch.Name != nil {
				channels = append(channels, *ch.Name)
			}
		}
		actions = append(actions, actionSummary{
			Name:           deref(a.Name),
			Label:          deref(a.Label),
			Description:    deref(a.Description),
			ConfigFields:   fields,
			OutputChannels: channels,
		})
	}

	resultJSON, _ := json.MarshalIndent(actions, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
