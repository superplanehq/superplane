package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// RegisterCanvasMutationTools registers canvas write/mutation MCP tools
func RegisterCanvasMutationTools(ctx context.Context, s *mcp.Server, apiClient *openapi_client.APIClient) error {
	// create_canvas tool
	createCanvasHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Name     string `json:"name"`
			YAMLSpec string `json:"yaml_spec"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleCreateCanvas(ctx, apiClient, args.Name, args.YAMLSpec)
	}

	s.AddTool(&mcp.Tool{
		Name:        "create_canvas",
		Description: "Create a new canvas from a YAML specification. The YAML must include apiVersion, kind, metadata (with name), and spec (with nodes and edges).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"description": "The name of the canvas to create"
				},
				"yaml_spec": {
					"type": "string",
					"description": "The complete YAML specification for the canvas including apiVersion, kind, metadata, and spec"
				}
			},
			"required": ["name", "yaml_spec"]
		}`),
	}, createCanvasHandler)

	// update_canvas tool
	updateCanvasHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID string `json:"canvas_id"`
			YAMLSpec string `json:"yaml_spec"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleUpdateCanvas(ctx, apiClient, args.CanvasID, args.YAMLSpec)
	}

	s.AddTool(&mcp.Tool{
		Name:        "update_canvas",
		Description: "Update an existing canvas with a new YAML specification. Creates a new version and publishes it automatically.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"canvas_id": {
					"type": "string",
					"description": "The ID of the canvas to update"
				},
				"yaml_spec": {
					"type": "string",
					"description": "The complete YAML specification for the canvas update"
				}
			},
			"required": ["canvas_id", "yaml_spec"]
		}`),
	}, updateCanvasHandler)

	// publish_canvas_version tool
	publishCanvasVersionHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID  string `json:"canvas_id"`
			VersionID string `json:"version_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handlePublishCanvasVersion(ctx, apiClient, args.CanvasID, args.VersionID)
	}

	s.AddTool(&mcp.Tool{
		Name:        "publish_canvas_version",
		Description: "Publish a draft canvas version to make it the active version.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"canvas_id": {
					"type": "string",
					"description": "The ID of the canvas"
				},
				"version_id": {
					"type": "string",
					"description": "The ID of the version to publish"
				}
			},
			"required": ["canvas_id", "version_id"]
		}`),
	}, publishCanvasVersionHandler)

	// validate_canvas tool
	validateCanvasHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			CanvasID  string `json:"canvas_id"`
			VersionID string `json:"version_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleValidateCanvas(ctx, apiClient, args.CanvasID, args.VersionID)
	}

	s.AddTool(&mcp.Tool{
		Name:        "validate_canvas",
		Description: "Validate a canvas version without publishing it. Returns any errors or warnings in the configuration.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"canvas_id": {
					"type": "string",
					"description": "The ID of the canvas"
				},
				"version_id": {
					"type": "string",
					"description": "The ID of the version to validate"
				}
			},
			"required": ["canvas_id", "version_id"]
		}`),
	}, validateCanvasHandler)

	return nil
}

// handleCreateCanvas creates a new canvas from YAML
func handleCreateCanvas(ctx context.Context, apiClient *openapi_client.APIClient, name string, yamlSpec string) (*mcp.CallToolResult, error) {
	// Parse YAML into Canvas resource
	resource, err := models.ParseCanvas([]byte(yamlSpec))
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Override name if provided
	if name != "" {
		if resource.Metadata == nil {
			resource.Metadata = &openapi_client.CanvasesCanvasMetadata{}
		}
		resource.Metadata.SetName(name)
	}

	// Build create request
	request := models.CreateCanvasRequestFromCanvas(*resource)

	// Set default auto-layout if not specified
	if resource.AutoLayout == nil {
		autoLayout := openapi_client.CanvasesCanvasAutoLayout{}
		autoLayout.SetAlgorithm(openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL)
		autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS)
		request.SetAutoLayout(autoLayout)
	}

	// Create canvas
	resp, _, err := apiClient.CanvasAPI.CanvasesCreateCanvas(ctx).Body(request).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create canvas: %w", err)
	}

	if resp == nil || resp.Canvas == nil || resp.Canvas.Metadata == nil {
		return nil, fmt.Errorf("invalid response from API")
	}

	canvas := resp.GetCanvas()
	metadata := canvas.GetMetadata()

	result := map[string]any{
		"id":      metadata.GetId(),
		"name":    metadata.GetName(),
		"message": fmt.Sprintf("Canvas %q created successfully", metadata.GetName()),
	}

	if metadata.HasCreatedAt() {
		result["created_at"] = metadata.GetCreatedAt()
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
	}, nil
}

// handleUpdateCanvas updates an existing canvas
func handleUpdateCanvas(ctx context.Context, apiClient *openapi_client.APIClient, canvasID string, yamlSpec string) (*mcp.CallToolResult, error) {
	// Parse YAML into Canvas resource
	resource, err := models.ParseCanvas([]byte(yamlSpec))
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	canvas := models.CanvasFromCanvas(*resource)

	// Find or create draft version
	targetVersionID, err := findOrCreateDraftVersion(ctx, apiClient, canvasID)
	if err != nil {
		return nil, err
	}

	// Update canvas version
	body := openapi_client.CanvasesUpdateCanvasVersionBody{}
	body.SetCanvas(canvas)
	body.SetVersionId(targetVersionID)

	// Set default auto-layout
	autoLayout := openapi_client.CanvasesCanvasAutoLayout{}
	autoLayout.SetAlgorithm(openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL)
	autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS)
	body.SetAutoLayout(autoLayout)

	response, _, err := apiClient.CanvasVersionAPI.
		CanvasesUpdateCanvasVersion2(ctx, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to update canvas version: %w", err)
	}

	version := response.GetVersion()

	// Check for errors
	if errText := formatNodeErrors(version); errText != "" {
		return nil, fmt.Errorf("canvas validation errors: %s", errText)
	}

	// Auto-publish the version
	_, _, publishErr := apiClient.CanvasVersionAPI.
		CanvasesPublishCanvasVersion(ctx, canvasID, targetVersionID).
		Body(map[string]any{}).
		Execute()
	if publishErr != nil {
		return nil, fmt.Errorf("draft was updated but publish failed: %w", publishErr)
	}

	metadata := version.GetMetadata()
	spec := version.GetSpec()

	result := map[string]any{
		"canvas_id":  metadata.GetCanvasId(),
		"version_id": metadata.GetId(),
		"state":      "published",
		"nodes":      len(spec.GetNodes()),
		"edges":      len(spec.GetEdges()),
		"message":    "Canvas updated and published successfully",
	}

	// Add warnings if any
	if warnText := formatNodeWarnings(version); warnText != "" {
		result["warnings"] = warnText
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
	}, nil
}

// handlePublishCanvasVersion publishes a draft canvas version
func handlePublishCanvasVersion(ctx context.Context, apiClient *openapi_client.APIClient, canvasID string, versionID string) (*mcp.CallToolResult, error) {
	// Publish the version
	_, _, err := apiClient.CanvasVersionAPI.
		CanvasesPublishCanvasVersion(ctx, canvasID, versionID).
		Body(map[string]any{}).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to publish canvas version: %w", err)
	}

	// Get the updated version details
	response, _, err := apiClient.CanvasVersionAPI.
		CanvasesDescribeCanvasVersion(ctx, canvasID, versionID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("published but failed to get version details: %w", err)
	}

	version := response.GetVersion()
	metadata := version.GetMetadata()

	result := map[string]any{
		"canvas_id":  canvasID,
		"version_id": versionID,
		"state":      metadata.GetState(),
		"message":    "Canvas version published successfully",
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
	}, nil
}

// handleValidateCanvas validates a canvas version
func handleValidateCanvas(ctx context.Context, apiClient *openapi_client.APIClient, canvasID string, versionID string) (*mcp.CallToolResult, error) {
	// Get version details
	response, _, err := apiClient.CanvasVersionAPI.
		CanvasesDescribeCanvasVersion(ctx, canvasID, versionID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to describe canvas version: %w", err)
	}

	version := response.GetVersion()
	metadata := version.GetMetadata()
	spec := version.GetSpec()

	result := map[string]any{
		"canvas_id":  canvasID,
		"version_id": versionID,
		"state":      metadata.GetState(),
		"nodes":      len(spec.GetNodes()),
		"edges":      len(spec.GetEdges()),
		"valid":      true,
	}

	// Check for errors
	if errText := formatNodeErrors(version); errText != "" {
		result["valid"] = false
		result["errors"] = errText
	}

	// Check for warnings
	if warnText := formatNodeWarnings(version); warnText != "" {
		result["warnings"] = warnText
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
	}, nil
}

// Helper functions

// findOrCreateDraftVersion finds the current user's draft version or creates one
func findOrCreateDraftVersion(ctx context.Context, apiClient *openapi_client.APIClient, canvasID string) (string, error) {
	// Try to find existing draft
	response, _, err := apiClient.CanvasVersionAPI.CanvasesListCanvasVersions(ctx, canvasID).Execute()
	if err != nil {
		return "", fmt.Errorf("failed to list canvas versions: %w", err)
	}

	for _, version := range response.GetVersions() {
		metadata := version.GetMetadata()
		if metadata.GetState() == openapi_client.CANVASESCANVASVERSIONSTATE_STATE_PUBLISHED {
			continue
		}

		versionID := strings.TrimSpace(metadata.GetId())
		if versionID != "" {
			return versionID, nil
		}
	}

	// No draft found, create one
	createResp, _, err := apiClient.CanvasVersionAPI.
		CanvasesCreateCanvasVersion(ctx, canvasID).
		Body(map[string]interface{}{}).
		Execute()
	if err != nil {
		return "", fmt.Errorf("failed to create draft version: %w", err)
	}

	if createResp.Version == nil || createResp.Version.Metadata == nil {
		return "", fmt.Errorf("draft version was not returned by the API")
	}

	versionID := strings.TrimSpace(createResp.Version.Metadata.GetId())
	if versionID == "" {
		return "", fmt.Errorf("draft version id was not returned by the API")
	}

	return versionID, nil
}

// formatNodeErrors formats node error messages
func formatNodeErrors(version openapi_client.CanvasesCanvasVersion) string {
	spec, ok := version.GetSpecOk()
	if !ok || spec == nil {
		return ""
	}

	var lines []string
	for _, node := range spec.GetNodes() {
		if !node.HasErrorMessage() {
			continue
		}
		msg := strings.TrimSpace(node.GetErrorMessage())
		if msg == "" {
			continue
		}
		id := node.GetId()
		name := strings.TrimSpace(node.GetName())
		if name == "" {
			name = id
		}
		lines = append(lines, fmt.Sprintf("node %s (%s): %s", id, name, msg))
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n")
}

// formatNodeWarnings formats node warning messages
func formatNodeWarnings(version openapi_client.CanvasesCanvasVersion) string {
	spec, ok := version.GetSpecOk()
	if !ok || spec == nil {
		return ""
	}

	var lines []string
	for _, node := range spec.GetNodes() {
		if !node.HasWarningMessage() {
			continue
		}
		msg := strings.TrimSpace(node.GetWarningMessage())
		if msg == "" {
			continue
		}
		id := node.GetId()
		name := strings.TrimSpace(node.GetName())
		if name == "" {
			name = id
		}
		lines = append(lines, fmt.Sprintf("node %s (%s): %s", id, name, msg))
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n")
}
