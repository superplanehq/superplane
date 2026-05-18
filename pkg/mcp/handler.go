package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	jsonRPCVersion = "2.0"
)

// JSON-RPC 2.0 error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Handler struct {
	jwt         *jwt.Signer
	registry    *registry.Registry
	staticToken string
}

func NewHandler(jwtSigner *jwt.Signer, reg *registry.Registry, staticToken string) *Handler {
	return &Handler{
		jwt:         jwtSigner,
		registry:    reg,
		staticToken: staticToken,
	}
}

// ServeHTTP implements the HTTP handler for MCP JSON-RPC 2.0 requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, nil, MethodNotFound, "Method not allowed", nil)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, nil, ParseError, "Failed to read request body", nil)
		return
	}

	// Parse JSON-RPC request
	var req jsonRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, nil, ParseError, "Invalid JSON-RPC request", nil)
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != jsonRPCVersion {
		h.writeError(w, req.ID, InvalidRequest, "Invalid JSON-RPC version", nil)
		return
	}

	// Extract and validate bearer token
	token, err := getBearerToken(r)
	if err != nil {
		h.writeError(w, req.ID, InvalidRequest, "Unauthorized: missing or invalid token", nil)
		return
	}

	var claims map[string]interface{}

	// Try static token first (backward compat), then JWT
	if h.staticToken != "" && token == h.staticToken {
		claims = map[string]interface{}{}
	} else {
		claims, err = h.jwt.ValidateAndGetClaims(token)
		if err != nil {
			h.writeError(w, req.ID, InvalidRequest, "Unauthorized: invalid token", nil)
			return
		}
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "claims", claims)
	ctx = context.WithValue(ctx, "bearer_token", token)

	// Route to appropriate handler
	var result interface{}
	switch req.Method {
	case "initialize":
		result, err = h.handleInitialize(ctx, req.Params)
	case "notifications/initialized":
		// No response needed for notifications
		return
	case "tools/list":
		result, err = h.handleToolsList(ctx, req.Params)
	case "tools/call":
		result, err = h.handleToolsCall(ctx, req.Params)
	default:
		h.writeError(w, req.ID, MethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
		return
	}

	if err != nil {
		log.Errorf("MCP handler error for method %s: %v", req.Method, err)
		h.writeError(w, req.ID, InternalError, err.Error(), nil)
		return
	}

	h.writeResponse(w, req.ID, result)
}

// handleInitialize handles the initialize request
func (h *Handler) handleInitialize(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]string{
			"name":    "superplane-mcp",
			"version": "1.0.0",
		},
	}, nil
}

// handleToolsList handles the tools/list request
func (h *Handler) handleToolsList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	tools := []map[string]interface{}{
		{
			"name":        "canvas_get",
			"description": "Retrieve a canvas (workflow) by ID in YAML or JSON format",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"canvas_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the canvas to retrieve",
					},
					"org_id": map[string]interface{}{
						"type":        "string",
						"description": "The organization ID that owns the canvas",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Output format: 'yaml' or 'json'",
						"enum":        []string{"yaml", "json"},
						"default":     "yaml",
					},
				},
				"required": []string{"canvas_id", "org_id"},
			},
		},
		{
			"name":        "canvas_list_versions",
			"description": "List all versions of a canvas",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"canvas_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the canvas",
					},
					"org_id": map[string]interface{}{
						"type":        "string",
						"description": "The organization ID that owns the canvas",
					},
				},
				"required": []string{"canvas_id", "org_id"},
			},
		},
		{
			"name":        "canvas_update",
			"description": "Update a canvas draft version with new YAML content",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"canvas_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the canvas to update",
					},
					"org_id": map[string]interface{}{
						"type":        "string",
						"description": "The organization ID that owns the canvas",
					},
					"yaml_content": map[string]interface{}{
						"type":        "string",
						"description": "Full canvas YAML content",
					},
				},
				"required": []string{"canvas_id", "org_id", "yaml_content"},
			},
		},
		{
			"name":        "integrations_list",
			"description": "List all integrations for an organization",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org_id": map[string]interface{}{
						"type":        "string",
						"description": "The organization ID",
					},
				},
				"required": []string{"org_id"},
			},
		},
		{
			"name":        "integrations_get",
			"description": "Get details for a specific integration instance",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"integration_id": map[string]interface{}{
						"type":        "string",
						"description": "The integration ID",
					},
					"org_id": map[string]interface{}{
						"type":        "string",
						"description": "The organization ID",
					},
				},
				"required": []string{"integration_id", "org_id"},
			},
		},
		{
			"name":        "index_search",
			"description": "Search the component registry for triggers and actions",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query string",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Optional filter: 'trigger' or 'action'",
						"enum":        []string{"trigger", "action"},
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "index_get_schema",
			"description": "Get full schema for a specific component (trigger or action)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"component_name": map[string]interface{}{
						"type":        "string",
						"description": "Component name (e.g. 'http', 'github.onPush')",
					},
				},
				"required": []string{"component_name"},
			},
		},
		{
			"name":        "runs_list",
			"description": "List recent runs for a canvas",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"canvas_id": map[string]interface{}{
						"type":        "string",
						"description": "The canvas ID",
					},
					"org_id": map[string]interface{}{
						"type":        "string",
						"description": "The organization ID",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Maximum number of runs to return (default 10)",
						"default":     10,
					},
				},
				"required": []string{"canvas_id", "org_id"},
			},
		},
		{
			"name":        "run_get",
			"description": "Get details for a specific canvas run",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"run_id": map[string]interface{}{
						"type":        "string",
						"description": "The run ID",
					},
					"canvas_id": map[string]interface{}{
						"type":        "string",
						"description": "The canvas ID",
					},
					"org_id": map[string]interface{}{
						"type":        "string",
						"description": "The organization ID",
					},
				},
				"required": []string{"run_id", "canvas_id", "org_id"},
			},
		},
	}

	return map[string]interface{}{
		"tools": tools,
	}, nil
}

// handleToolsCall handles the tools/call request
func (h *Handler) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var callReq struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(params, &callReq); err != nil {
		return nil, fmt.Errorf("invalid tool call params: %w", err)
	}

	switch callReq.Name {
	case "canvas_get":
		return handleCanvasGet(ctx, h.registry, callReq.Arguments)
	case "canvas_list_versions":
		return handleCanvasListVersions(ctx, callReq.Arguments)
	case "canvas_update":
		return handleCanvasUpdate(ctx, callReq.Arguments)
	case "integrations_list":
		return handleIntegrationsList(ctx, callReq.Arguments)
	case "integrations_get":
		return handleIntegrationsGet(ctx, callReq.Arguments)
	case "index_search":
		return handleIndexSearch(ctx, h.registry, callReq.Arguments)
	case "index_get_schema":
		return handleIndexGetSchema(ctx, h.registry, callReq.Arguments)
	case "runs_list":
		return handleRunsList(ctx, callReq.Arguments)
	case "run_get":
		return handleRunGet(ctx, callReq.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", callReq.Name)
	}
}

func (h *Handler) writeResponse(w http.ResponseWriter, id interface{}, result interface{}) {
	resp := jsonRPCResponse{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) writeError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	resp := jsonRPCResponse{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func getBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header not found")
	}

	const prefix = "Bearer "
	if len(authHeader) < len(prefix) || authHeader[:len(prefix)] != prefix {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return authHeader[len(prefix):], nil
}
