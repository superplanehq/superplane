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
	jwt      *jwt.Signer
	registry *registry.Registry
}

func NewHandler(jwtSigner *jwt.Signer, reg *registry.Registry) *Handler {
	return &Handler{
		jwt:      jwtSigner,
		registry: reg,
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

	// Extract and validate JWT token
	token, err := getBearerToken(r)
	if err != nil {
		h.writeError(w, req.ID, InvalidRequest, "Unauthorized: missing or invalid token", nil)
		return
	}

	claims, err := h.jwt.ValidateAndGetClaims(token)
	if err != nil {
		h.writeError(w, req.ID, InvalidRequest, "Unauthorized: invalid token", nil)
		return
	}

	// Extract org_id and user_id from claims
	ctx := context.Background()
	ctx = context.WithValue(ctx, "claims", claims)

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
	case "integrations_list":
		return handleIntegrationsList(ctx, callReq.Arguments)
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
