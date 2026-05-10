package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// RegisterEventTools registers event-related MCP tools
func RegisterEventTools(ctx context.Context, s *mcp.Server, apiClient *openapi_client.APIClient) error {
	// emit_event tool
	emitEventHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID string                 `json:"canvas_id"`
			NodeID   string                 `json:"node_id"`
			Channel  string                 `json:"channel"`
			Data     map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		if args.Channel == "" {
			args.Channel = "default"
		}
		return handleEmitEvent(ctx, apiClient, args.CanvasID, args.NodeID, args.Channel, args.Data)
	}

	s.AddTool(&mcp.Tool{
		Name:        "emit_event",
		Description: "Emit an event to a canvas node trigger. This starts a new execution flow.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"canvas_id":{"type":"string","description":"The ID of the canvas"},"node_id":{"type":"string","description":"The ID of the node to emit event to"},"channel":{"type":"string","description":"Event channel (defaults to 'default')"},"data":{"type":"object","description":"Event data as a JSON object"}},"required":["canvas_id","node_id","data"]}`),
	}, emitEventHandler)

	// list_events tool
	listEventsHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID string `json:"canvas_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleListEvents(ctx, apiClient, args.CanvasID)
	}

	s.AddTool(&mcp.Tool{
		Name:        "list_events",
		Description: "List recent root events for a canvas. Returns event ID, node, state, and timestamp.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"canvas_id":{"type":"string","description":"The ID of the canvas"}},"required":["canvas_id"]}`),
	}, listEventsHandler)

	return nil
}

// handleEmitEvent emits an event for a canvas node
func handleEmitEvent(ctx context.Context, apiClient *openapi_client.APIClient, canvasID, nodeID, channel string, data map[string]interface{}) (*mcp.CallToolResult, error) {
	body := openapi_client.NewCanvasesEmitNodeEventBody()
	body.SetChannel(channel)
	body.SetData(data)

	response, _, err := apiClient.CanvasNodeAPI.CanvasesEmitNodeEvent(ctx, canvasID, nodeID).Body(*body).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to emit event: %w", err)
	}

	result := map[string]any{
		"event_id":  response.GetEventId(),
		"canvas_id": canvasID,
		"node_id":   nodeID,
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
	}, nil
}

// handleListEvents lists recent canvas events
func handleListEvents(ctx context.Context, apiClient *openapi_client.APIClient, canvasID string) (*mcp.CallToolResult, error) {
	response, _, err := apiClient.CanvasEventAPI.CanvasesListCanvasEvents(ctx, canvasID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	events := response.GetEvents()
	results := make([]map[string]any, 0, len(events))

	for _, event := range events {
		result := map[string]any{
			"id":      event.GetId(),
			"node_id": event.GetNodeId(),
		}

		if event.HasChannel() {
			result["channel"] = event.GetChannel()
		}

		if event.HasCustomName() {
			result["custom_name"] = event.GetCustomName()
		}

		if event.HasCreatedAt() {
			result["created_at"] = event.GetCreatedAt()
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
