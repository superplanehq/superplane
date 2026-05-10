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

	// describe_action tool
	describeActionHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleDescribeAction(ctx, apiClient, args.Name)
	}

	s.AddTool(&mcp.Tool{
		Name:        "describe_action",
		Description: `Get detailed docs for an action component. Returns full configuration schema with field types, nested objects, required fields, defaults, select options, and value format hints (e.g. secret-key fields need {secretName: "SECRET_NAME"} map format, list fields with object items need [{name: "x", value: "y"}] format). Use before creating canvases with complex components.`,
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"Action name (e.g. http, ssh, approval, if, timeGate)"}},"required":["name"]}`),
	}, describeActionHandler)

	// describe_trigger tool
	describeTriggerHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return handleDescribeTrigger(ctx, apiClient, args.Name)
	}

	s.AddTool(&mcp.Tool{
		Name:        "describe_trigger",
		Description: "Get detailed docs for a trigger component including full configuration schema.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"Trigger name (e.g. webhook, schedule, start)"}},"required":["name"]}`),
	}, describeTriggerHandler)

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

	triggers := make([]map[string]any, 0, len(response.Triggers))
	for _, t := range response.Triggers {
		trigger := map[string]any{
			"name":        deref(t.Name),
			"label":       deref(t.Label),
			"description": deref(t.Description),
		}
		fields := buildConfigSchema(t.Configuration)
		if len(fields) > 0 {
			trigger["configuration_schema"] = fields
		}
		triggers = append(triggers, trigger)
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

	actions := make([]map[string]any, 0, len(response.Actions))
	for _, a := range response.Actions {
		action := map[string]any{
			"name":        deref(a.Name),
			"label":       deref(a.Label),
			"description": deref(a.Description),
		}
		channels := make([]string, 0, len(a.OutputChannels))
		for _, ch := range a.OutputChannels {
			if ch.Name != nil {
				channels = append(channels, *ch.Name)
			}
		}
		if len(channels) > 0 {
			action["output_channels"] = channels
		}
		fields := buildConfigSchema(a.Configuration)
		if len(fields) > 0 {
			action["configuration_schema"] = fields
		}
		actions = append(actions, action)
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

type configFieldSchema struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Label        string   `json:"label,omitempty"`
	Description  string   `json:"description,omitempty"`
	Required     bool     `json:"required"`
	DefaultValue string   `json:"default,omitempty"`
	Options      []string `json:"options,omitempty"`
	Placeholder  string   `json:"placeholder,omitempty"`
}

func buildConfigSchema(fields []openapi_client.ConfigurationField) []configFieldSchema {
	result := make([]configFieldSchema, 0, len(fields))
	for _, f := range fields {
		schema := configFieldSchema{
			Name:        deref(f.Name),
			Type:        deref(f.Type),
			Label:       deref(f.Label),
			Description: deref(f.Description),
			Placeholder: deref(f.Placeholder),
		}
		if f.Required != nil {
			schema.Required = *f.Required
		}
		if f.DefaultValue != nil {
			schema.DefaultValue = *f.DefaultValue
		}
		// Include select options if available
		if f.TypeOptions != nil && f.TypeOptions.Select != nil {
			for _, opt := range f.TypeOptions.Select.Options {
				if opt.Value != nil {
					schema.Options = append(schema.Options, *opt.Value)
				}
			}
		}
		result = append(result, schema)
	}
	return result
}

// deepFieldSchema is a rich field description including nested objects and format hints
type deepFieldSchema struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Label        string            `json:"label,omitempty"`
	Description  string            `json:"description,omitempty"`
	Required     bool              `json:"required"`
	DefaultValue string            `json:"default,omitempty"`
	Options      []string          `json:"options,omitempty"`
	Placeholder  string            `json:"placeholder,omitempty"`
	FormatHint   string            `json:"format_hint,omitempty"`
	NestedSchema []deepFieldSchema `json:"nested_schema,omitempty"`
	ItemSchema   []deepFieldSchema `json:"item_schema,omitempty"`
}

func buildDeepConfigSchema(fields []openapi_client.ConfigurationField) []deepFieldSchema {
	result := make([]deepFieldSchema, 0, len(fields))
	for _, f := range fields {
		schema := deepFieldSchema{
			Name:        deref(f.Name),
			Type:        deref(f.Type),
			Label:       deref(f.Label),
			Description: deref(f.Description),
			Placeholder: deref(f.Placeholder),
		}
		if f.Required != nil {
			schema.Required = *f.Required
		}
		if f.DefaultValue != nil {
			schema.DefaultValue = *f.DefaultValue
		}

		// Add format hints for special types
		switch deref(f.Type) {
		case "secret-key":
			schema.FormatHint = `Value must be a map: {secretName: "YOUR_SECRET_NAME"}. The secret must exist in SuperPlane org secrets.`
		case "number":
			schema.FormatHint = "Value must be a string representation of a number, e.g. \"30\""
		case "boolean":
			schema.FormatHint = `Value must be string "true" or "false"`
		}

		if f.TypeOptions != nil {
			// Select options
			if f.TypeOptions.Select != nil {
				for _, opt := range f.TypeOptions.Select.Options {
					if opt.Value != nil {
						schema.Options = append(schema.Options, *opt.Value)
					}
				}
			}
			// Nested object schema
			if f.TypeOptions.Object != nil && len(f.TypeOptions.Object.Schema) > 0 {
				schema.NestedSchema = buildDeepConfigSchema(f.TypeOptions.Object.Schema)
			}
			// List item schema
			if f.TypeOptions.List != nil && f.TypeOptions.List.ItemDefinition != nil {
				itemDef := f.TypeOptions.List.ItemDefinition
				if itemDef.Schema != nil && len(itemDef.Schema) > 0 {
					schema.ItemSchema = buildDeepConfigSchema(itemDef.Schema)
					schema.FormatHint = "Value is a list of objects. Each item has the fields described in item_schema."
				}
			}
		}
		result = append(result, schema)
	}
	return result
}

func handleDescribeAction(ctx context.Context, apiClient *openapi_client.APIClient, name string) (*mcp.CallToolResult, error) {
	response, _, err := apiClient.ActionAPI.ActionsDescribeAction(ctx, name).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to describe action %q: %w", name, err)
	}

	a := response.GetAction()
	result := map[string]any{
		"name":        deref(a.Name),
		"label":       deref(a.Label),
		"description": deref(a.Description),
	}

	channels := make([]string, 0, len(a.OutputChannels))
	for _, ch := range a.OutputChannels {
		if ch.Name != nil {
			channels = append(channels, *ch.Name)
		}
	}
	if len(channels) > 0 {
		result["output_channels"] = channels
	}

	fields := buildDeepConfigSchema(a.Configuration)
	if len(fields) > 0 {
		result["configuration_schema"] = fields
	}

	if a.ExampleOutput != nil {
		result["example_output"] = a.ExampleOutput
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil
}

func handleDescribeTrigger(ctx context.Context, apiClient *openapi_client.APIClient, name string) (*mcp.CallToolResult, error) {
	response, _, err := apiClient.TriggerAPI.TriggersDescribeTrigger(ctx, name).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to describe trigger %q: %w", name, err)
	}

	t := response.GetTrigger()
	result := map[string]any{
		"name":        deref(t.Name),
		"label":       deref(t.Label),
		"description": deref(t.Description),
	}

	fields := buildDeepConfigSchema(t.Configuration)
	if len(fields) > 0 {
		result["configuration_schema"] = fields
	}

	if t.ExampleData != nil {
		result["example_data"] = t.ExampleData
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil
}
