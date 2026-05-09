package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	"gopkg.in/yaml.v3"
)

// RegisterCanvasTools registers canvas-related MCP tools
func RegisterCanvasTools(ctx context.Context, s *mcp.Server, apiClient *openapi_client.APIClient) error {
	// list_canvases tool
	listCanvasesHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListCanvases(ctx, apiClient)
	}

	s.AddTool(&mcp.Tool{
		Name:        "list_canvases",
		Description: "List all canvases in the organization. Returns canvas ID and name.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, listCanvasesHandler)

	// describe_canvas tool
	describeCanvasHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID string `json:"canvas_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleDescribeCanvas(ctx, apiClient, args.CanvasID)
	}

	s.AddTool(&mcp.Tool{
		Name:        "describe_canvas",
		Description: "Get full details of a specific canvas including its current version YAML configuration.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"canvas_id":{"type":"string","description":"The ID of the canvas to describe"}},"required":["canvas_id"]}`),
	}, describeCanvasHandler)

	return nil
}

// handleListCanvases lists all canvases
func handleListCanvases(ctx context.Context, apiClient *openapi_client.APIClient) (*mcp.CallToolResult, error) {
	response, _, err := apiClient.CanvasAPI.CanvasesListCanvases(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list canvases: %w", err)
	}

	canvases := response.GetCanvases()
	results := make([]map[string]any, 0, len(canvases))

	for _, canvas := range canvases {
		metadata := canvas.GetMetadata()

		result := map[string]any{
			"id":   metadata.GetId(),
			"name": metadata.GetName(),
		}

		if metadata.HasCreatedAt() {
			result["created_at"] = metadata.GetCreatedAt()
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

// handleDescribeCanvas describes a single canvas with full details
func handleDescribeCanvas(ctx context.Context, apiClient *openapi_client.APIClient, canvasID string) (*mcp.CallToolResult, error) {
	response, _, err := apiClient.CanvasAPI.CanvasesDescribeCanvas(ctx, canvasID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to describe canvas: %w", err)
	}

	canvas := response.GetCanvas()
	metadata := canvas.GetMetadata()
	spec := canvas.GetSpec()

	result := map[string]any{
		"id":   metadata.GetId(),
		"name": metadata.GetName(),
	}

	if metadata.HasCreatedAt() {
		result["created_at"] = metadata.GetCreatedAt()
	}

	if metadata.HasUpdatedAt() {
		result["updated_at"] = metadata.GetUpdatedAt()
	}

	// Include the full YAML spec
	specYAML, err := yaml.Marshal(spec)
	if err == nil {
		result["spec_yaml"] = string(specYAML)
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
	}, nil
}
