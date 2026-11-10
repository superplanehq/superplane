package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/logging"
)

const maxJSONRPCBytes = 1 << 20

var supportedProtocolVersions = []string{
	"2024-11-05",
	"2025-03-26",
	"2025-06-18",
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ToolResult struct {
	Content []map[string]any `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func ParseJSONRPC(body io.Reader) (*JSONRPCRequest, error) {
	limited := io.LimitReader(body, maxJSONRPCBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(raw) > maxJSONRPCBytes {
		return nil, fmt.Errorf("request is too large")
	}
	var message JSONRPCRequest
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil, err
	}
	if message.JSONRPC != "2.0" {
		return nil, fmt.Errorf("jsonrpc must be 2.0")
	}
	return &message, nil
}

func HandleJSONRPC(ctx context.Context, runtime *Runtime, claims *AccessClaims, message *JSONRPCRequest) *JSONRPCResponse {
	if message.Method == "notifications/initialized" || message.Method == "notifications/cancelled" {
		return nil
	}
	if message.ID == nil {
		return nil
	}
	switch message.Method {
	case "initialize":
		return rpcResult(message.ID, initializeResult(message.Params))
	case "ping":
		return rpcResult(message.ID, map[string]any{})
	case "tools/list":
		return rpcResult(message.ID, map[string]any{"tools": Tools()})
	case "tools/call":
		return callTool(ctx, runtime, claims, message)
	case "resources/list":
		return rpcResult(message.ID, map[string]any{"resources": []any{}})
	case "resources/templates/list":
		return rpcResult(message.ID, map[string]any{"resourceTemplates": []any{}})
	case "prompts/list":
		return rpcResult(message.ID, map[string]any{"prompts": []any{}})
	case "logging/setLevel":
		return rpcResult(message.ID, map[string]any{})
	default:
		return rpcError(message.ID, -32601, "Unknown method: "+message.Method)
	}
}

func initializeResult(params json.RawMessage) map[string]any {
	version := "2024-11-05"
	var parsed struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(params, &parsed); err == nil && slices.Contains(supportedProtocolVersions, parsed.ProtocolVersion) {
		version = parsed.ProtocolVersion
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities":    map[string]any{"tools": map[string]any{}},
		"serverInfo":      ServerInfo(),
	}
}

func callTool(ctx context.Context, runtime *Runtime, claims *AccessClaims, message *JSONRPCRequest) (response *JSONRPCResponse) {
	toolName := "unknown"
	targetType, targetID := "", ""
	defer func() {
		outcome := "error"
		if response != nil && response.Error == nil {
			if result, ok := response.Result.(ToolResult); ok && !result.IsError {
				outcome = "success"
			}
		}
		identity := logging.SecurityIdentity{UserID: claims.UserID.String(), OrganizationID: claims.OrgID.String()}
		if known := logging.SecurityIdentityFrom(ctx); known != nil {
			identity = *known
		}
		logging.SecurityEvent(logging.WithSecurityIdentity(ctx, &identity), "mcp.tool_call", outcome, targetType, targetID, log.Fields{"tool_name": toolName})
	}()
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil || strings.TrimSpace(params.Name) == "" {
		return rpcError(message.ID, -32602, "tool name is required")
	}
	for _, tool := range Tools() {
		if tool.Name == params.Name {
			toolName = tool.Name
			break
		}
	}
	args := map[string]any{}
	if len(params.Arguments) > 0 {
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return rpcError(message.ID, -32602, "tool arguments must be an object")
		}
	}
	// Only UUID targets are logged. Task URLs, keys, and other argument text stay private.
	for _, key := range []string{"task", "workspace"} {
		if value, ok := args[key].(string); ok {
			if id, err := uuid.Parse(value); err == nil {
				targetType, targetID = key, id.String()
				break
			}
		}
	}
	result, err := runtime.CallTool(ctx, claims, params.Name, args)
	if err != nil {
		return rpcResult(message.ID, ToolResult{
			Content: []map[string]any{{"type": "text", "text": err.Error()}},
			IsError: true,
		})
	}
	return rpcResult(message.ID, result)
}

func rpcResult(id any, result any) *JSONRPCResponse {
	return &JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func rpcError(id any, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{JSONRPC: "2.0", ID: id, Error: &JSONRPCError{Code: code, Message: message}}
}

func TextResult(text string) ToolResult {
	return ToolResult{Content: []map[string]any{{"type": "text", "text": text}}}
}

func ToolError(message string) error {
	return fmt.Errorf("%s", message)
}
