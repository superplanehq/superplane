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
		Name: "create_canvas",
		Description: `Create a new canvas (workflow) from a YAML specification.

YAML Schema:
` + "```" + `yaml
apiVersion: v1
kind: Canvas
metadata:
  name: <canvas-name>
spec:
  nodes:
    - id: <unique-node-id>        # lowercase-kebab-case
      name: <Display Name>
      type: TYPE_TRIGGER            # TYPE_TRIGGER | TYPE_ACTION
      component: <component-name>   # e.g. webhook, http, ssh, approval, filter, if, noop, timeGate, wait, merge, upsertMemory, readMemory, deleteMemory, addMemory, updateMemory, schedule, start, sendEmail, graphql
      configuration:                # component-specific config (all values must be strings)
        key1: "value1"
      integration:                  # only for integration-bound components
        name: <integration-name>
  edges:
    - sourceId: <source-node-id>
      targetId: <target-node-id>
      channel: <output-channel>     # e.g. default, success, failure, approved, rejected, true, false, found, notFound, deleted
` + "```" + `

Key rules:
- node.type must be TYPE_TRIGGER or TYPE_ACTION (not "trigger" or "action")
- node.component is the component name from list_triggers/list_actions
- All configuration values must be strings, including numbers (e.g. "30" not 30)
- edge.channel specifies which output of the source node triggers the target
- Common output channels: webhook/schedule/start triggers use "default"; http uses "success"/"failure"; approval uses "approved"/"rejected"; if uses "true"/"false"; readMemory uses "found"/"notFound"; deleteMemory uses "deleted"
- For http action: set method, url, headers (as formData with key/value pairs), body, timeoutSeconds (string, max "30"), successCodes (string, e.g. "200,201")
- For ssh action: set host, port, username, command; authentication requires privateKey as a map: {secretName: "SECRET_NAME"}
- For approval action: set message (string), approvalType (e.g. "TYPE_ANYONE")
- For timeGate action: set activeDays (comma-separated, e.g. "monday,tuesday,..."), timeRange (e.g. "09:00-17:00"), timezone (numeric offset string, e.g. "0" for UTC)
- For if action: set expression (e.g. "data.status == 'ok'")
- For memory actions (upsertMemory/readMemory/deleteMemory): set key (the lookup key expression), and for upsert set value
- For filter action: set expression

Example - webhook trigger connected to an approval gate:
` + "```" + `yaml
apiVersion: v1
kind: Canvas
metadata:
  name: my-workflow
spec:
  nodes:
    - id: webhook-trigger
      name: Incoming Request
      type: TYPE_TRIGGER
      component: webhook
      configuration:
        authentication: signature
        signatureHeader: X-Hub-Signature-256
    - id: approval-gate
      name: Require Approval
      type: TYPE_ACTION
      component: approval
      configuration:
        message: "A new request needs approval"
        approvalType: TYPE_ANYONE
    - id: on-approved
      name: Approved
      type: TYPE_ACTION
      component: noop
    - id: on-rejected
      name: Rejected
      type: TYPE_ACTION
      component: noop
  edges:
    - sourceId: webhook-trigger
      targetId: approval-gate
      channel: default
    - sourceId: approval-gate
      targetId: on-approved
      channel: approved
    - sourceId: approval-gate
      targetId: on-rejected
      channel: rejected
` + "```",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"description": "The name of the canvas to create"
				},
				"yaml_spec": {
					"type": "string",
					"description": "The complete YAML specification for the canvas"
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

	// Check for validation errors on newly created canvas
	canvasID := metadata.GetId()
	spec := canvas.GetSpec()
	result["nodes"] = len(spec.GetNodes())
	result["edges"] = len(spec.GetEdges())

	versionsResp, _, versionsErr := apiClient.CanvasVersionAPI.CanvasesListCanvasVersions(ctx, canvasID).Execute()
	if versionsErr == nil && len(versionsResp.GetVersions()) > 0 {
		latestVersion := versionsResp.GetVersions()[0]
		if errText := formatNodeErrors(latestVersion); errText != "" {
			// Truncate to keep response small for SSE transport
			if len(errText) > 2000 {
				errText = errText[:2000] + "\n... (truncated, use describe_canvas for full errors)"
			}
			result["validation_errors"] = errText
			result["message"] = fmt.Sprintf("Canvas %q created but has validation errors", metadata.GetName())
		}
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

	// Check for errors — return as result content, not as tool error
	nodeErrors := formatNodeErrors(version)

	// Auto-publish the version (even if there are validation warnings)
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

	// Include validation errors as part of result (not as tool error)
	if nodeErrors != "" {
		// Truncate to keep response small for SSE transport
		if len(nodeErrors) > 2000 {
			nodeErrors = nodeErrors[:2000] + "\n... (truncated, use describe_canvas for full errors)"
		}
		result["validation_errors"] = nodeErrors
		result["message"] = "Canvas updated and published, but has validation errors"
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
