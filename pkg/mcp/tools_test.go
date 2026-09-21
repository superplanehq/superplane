package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListToolsReturnsJSONTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json, text/event-stream", r.Header.Get("Accept"))
		assert.Equal(t, "Bearer secret", r.Header.Get("Authorization"))

		var payload rpcRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		switch payload.Method {
		case "initialize":
			w.Header().Set(mcpSessionHeader, "session-1")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]any{
					"protocolVersion": mcpProtocolVersion,
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "example"},
				},
			})
		case "notifications/initialized":
			assert.Equal(t, "session-1", r.Header.Get(mcpSessionHeader))
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			assert.Equal(t, "session-1", r.Header.Get(mcpSessionHeader))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      2,
				"result": map[string]any{
					"tools": []map[string]any{
						{"name": "search", "description": "Search the catalog."},
						{"name": "create_issue", "description": "Create an issue."},
					},
				},
			})
		default:
			t.Fatalf("unexpected method %s", payload.Method)
		}
	}))
	t.Cleanup(server.Close)

	tools, err := ListTools(context.Background(), server.Client(), server.URL, map[string]string{
		"Authorization": "Bearer secret",
	})
	require.NoError(t, err)
	require.Equal(t, []Tool{
		{Name: "search", Description: "Search the catalog."},
		{Name: "create_issue", Description: "Create an issue."},
	}, tools)
}

func TestListToolsReadsSSEPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload rpcRequest
		require.NoError(t, json.Unmarshal(body, &payload))
		w.Header().Set("Content-Type", "text/event-stream")
		switch payload.Method {
		case "initialize":
			_, _ = io.WriteString(w, "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\n\n")
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			_, _ = io.WriteString(w, "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"tools\":[{\"name\":\"ping\",\"description\":\"Ping the server.\"}]}}\n\n")
		default:
			t.Fatalf("unexpected method %s", payload.Method)
		}
	}))
	t.Cleanup(server.Close)

	tools, err := ListTools(context.Background(), server.Client(), server.URL, nil)
	require.NoError(t, err)
	require.Equal(t, []Tool{{Name: "ping", Description: "Ping the server."}}, tools)
}

func TestListToolsReturnsEmptyList(t *testing.T) {
	server := httptest.NewServer(jsonRPCToolsHandler(t, []map[string]any{}))
	t.Cleanup(server.Close)

	tools, err := ListTools(context.Background(), server.Client(), server.URL, nil)
	require.NoError(t, err)
	assert.Empty(t, tools)
}

func TestListToolsRejectsNonJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "<html>not json</html>")
	}))
	t.Cleanup(server.Close)

	_, err := ListTools(context.Background(), server.Client(), server.URL, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "the MCP server did not return JSON")
}

func jsonRPCToolsHandler(t *testing.T, tools []map[string]any) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload rpcRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		switch payload.Method {
		case "initialize":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{}})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      2,
				"result":  map[string]any{"tools": tools},
			})
		default:
			t.Fatalf("unexpected method %s", payload.Method)
		}
	})
}
