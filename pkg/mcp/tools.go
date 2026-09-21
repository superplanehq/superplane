package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	mcpProtocolVersion = "2025-03-26"
	mcpSessionHeader   = "Mcp-Session-Id"
)

// Tool is one MCP tools/list entry. Schema is omitted on purpose.
type Tool struct {
	Name        string
	Description string
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type initializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      clientInfo     `json:"clientInfo"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type toolsListResult struct {
	Tools []toolPayload `json:"tools"`
}

type toolPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListTools initializes a Streamable HTTP MCP session and returns tools/list.
func ListTools(ctx context.Context, httpClient HTTPDoer, mcpURL string, headers map[string]string) ([]Tool, error) {
	if strings.TrimSpace(mcpURL) == "" {
		return nil, fmt.Errorf("MCP URL is required")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	rpcCtx, cancel := TimeoutContext(ctx)
	defer cancel()

	sessionID, err := initializeMCPSession(rpcCtx, httpClient, mcpURL, headers)
	if err != nil {
		return nil, err
	}
	_ = postMCPNotification(rpcCtx, httpClient, mcpURL, headers, sessionID, "notifications/initialized")

	var result toolsListResult
	if _, err := postMCP(rpcCtx, httpClient, mcpURL, headers, sessionID, rpcRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
		Params:  map[string]any{},
	}, &result); err != nil {
		return nil, err
	}

	tools := make([]Tool, 0, len(result.Tools))
	for _, tool := range result.Tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		tools = append(tools, Tool{Name: name, Description: strings.TrimSpace(tool.Description)})
	}
	return tools, nil
}

func initializeMCPSession(ctx context.Context, httpClient HTTPDoer, mcpURL string, headers map[string]string) (string, error) {
	sessionID, err := postMCP(ctx, httpClient, mcpURL, headers, "", rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: initializeParams{
			ProtocolVersion: mcpProtocolVersion,
			Capabilities:    map[string]any{},
			ClientInfo:      clientInfo{Name: ClientName, Version: "1.0"},
		},
	}, nil)
	return sessionID, err
}

func postMCPNotification(ctx context.Context, httpClient HTTPDoer, mcpURL string, headers map[string]string, sessionID, method string) error {
	_, err := postMCP(ctx, httpClient, mcpURL, headers, sessionID, rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
	}, nil)
	return err
}

func postMCP(
	ctx context.Context,
	httpClient HTTPDoer,
	mcpURL string,
	headers map[string]string,
	sessionID string,
	payload rpcRequest,
	dest any,
) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mcpURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", mcpProtocolVersion)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	if sessionID != "" {
		req.Header.Set(mcpSessionHeader, sessionID)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call MCP server: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, MaxMetadataBytes)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("the MCP server returned HTTP %d", resp.StatusCode)
	}

	nextSession := strings.TrimSpace(resp.Header.Get(mcpSessionHeader))
	if payload.Method == "notifications/initialized" {
		return nextSession, nil
	}
	if dest == nil && payload.Method != "initialize" {
		return nextSession, nil
	}

	decoded, err := decodeMCPBody(resp.Header.Get("Content-Type"), raw)
	if err != nil {
		return "", err
	}
	var envelope rpcResponse
	if err := json.Unmarshal(decoded, &envelope); err != nil {
		return "", fmt.Errorf("the MCP server did not return JSON")
	}
	if envelope.Error != nil {
		message := strings.TrimSpace(envelope.Error.Message)
		if message == "" {
			message = "the MCP server returned an error"
		}
		return "", fmt.Errorf("%s", message)
	}
	if dest == nil {
		return nextSession, nil
	}
	if len(envelope.Result) == 0 {
		return nextSession, nil
	}
	if err := json.Unmarshal(envelope.Result, dest); err != nil {
		return "", fmt.Errorf("the MCP server did not return JSON")
	}
	return nextSession, nil
}

func decodeMCPBody(contentType string, body []byte) ([]byte, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, fmt.Errorf("the MCP server did not return JSON")
	}
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") {
		return sseJSONPayload(body)
	}
	trimmed := bytes.TrimSpace(body)
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("the MCP server did not return JSON")
	}
	return trimmed, nil
}

func sseJSONPayload(body []byte) ([]byte, error) {
	var payload []byte
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload = []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
	}
	if len(payload) == 0 || !json.Valid(payload) {
		return nil, fmt.Errorf("the MCP server did not return JSON")
	}
	return payload, nil
}
