package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// TestMCPServerIntegration tests the full MCP server flow end-to-end
func TestMCPServerIntegration(t *testing.T) {
	// Create a test HTTP server to mock the SuperPlane API
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases":
			// Mock list_canvases response
			response := `{
				"canvases": [
					{
						"metadata": {
							"id": "canvas-001",
							"name": "test-canvas",
							"createdAt": "2025-01-15T10:00:00Z"
						}
					},
					{
						"metadata": {
							"id": "canvas-002",
							"name": "another-canvas",
							"createdAt": "2025-01-16T10:00:00Z"
						}
					}
				]
			}`
			_, _ = w.Write([]byte(response))

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-1/events":
			// Mock emit_event response
			response := `{
				"eventId": "event-001"
			}`
			_, _ = w.Write([]byte(response))

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(apiServer.Close)

	// Create API client pointing to the test server
	apiConfig := openapi_client.NewConfiguration()
	apiConfig.Servers = openapi_client.ServerConfigurations{{URL: apiServer.URL}}
	apiClient := openapi_client.NewAPIClient(apiConfig)

	// Create in-memory transports for testing
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	// Create MCP server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "superplane-test",
			Version: "test",
		},
		nil,
	)

	// Register tools
	ctx := context.Background()
	err := registerTools(ctx, server, apiClient)
	require.NoError(t, err, "failed to register tools")

	// Start server in a goroutine
	serverDone := make(chan error, 1)
	go func() {
		_, err := server.Connect(ctx, serverTransport, nil)
		serverDone <- err
	}()

	// Create MCP client
	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "test-client",
			Version: "test",
		},
		nil,
	)

	// Connect client
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err, "client failed to connect")
	defer session.Close()

	// Give the server time to initialize
	time.Sleep(100 * time.Millisecond)

	// Test 1: Initialize handshake (happens automatically during Connect)
	t.Run("handshake", func(t *testing.T) {
		// The handshake is implicit in Connect, so we just verify the connection works
		require.NotNil(t, client)
	})

	// Test 2: List tools
	t.Run("list_tools", func(t *testing.T) {
		result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		require.NoError(t, err, "failed to list tools")
		require.NotNil(t, result)

		// Verify expected tools are present
		toolNames := make(map[string]bool)
		for _, tool := range result.Tools {
			toolNames[tool.Name] = true
		}

		// Check for key tools from each category
		expectedTools := []string{
			"list_canvases",
			"describe_canvas",
			"emit_event",
			"list_events",
			"list_executions",
			"describe_execution",
			"list_integrations",
			"list_secrets",
			"create_canvas",
			"update_canvas",
		}

		for _, toolName := range expectedTools {
			require.True(t, toolNames[toolName], "expected tool %s not found", toolName)
		}
	})

	// Test 3: Call list_canvases tool
	t.Run("call_list_canvases", func(t *testing.T) {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "list_canvases",
			Arguments: map[string]any{},
		})
		require.NoError(t, err, "failed to call list_canvases")
		require.NotNil(t, result)
		require.Len(t, result.Content, 1, "expected one content item")

		// Parse the response
		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok, "expected text content")
		require.NotEmpty(t, textContent.Text)

		var canvases []map[string]any
		err = json.Unmarshal([]byte(textContent.Text), &canvases)
		require.NoError(t, err, "failed to parse canvases JSON")
		require.Len(t, canvases, 2, "expected 2 canvases")
		require.Equal(t, "canvas-001", canvases[0]["id"])
		require.Equal(t, "test-canvas", canvases[0]["name"])
		require.Equal(t, "canvas-002", canvases[1]["id"])
		require.Equal(t, "another-canvas", canvases[1]["name"])
	})

	// Test 4: Call emit_event tool
	t.Run("call_emit_event", func(t *testing.T) {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "emit_event",
			Arguments: map[string]any{
				"canvas_id": "canvas-001",
				"node_id":   "node-1",
				"data": map[string]any{
					"foo": "bar",
					"baz": 42,
				},
			},
		})
		require.NoError(t, err, "failed to call emit_event")
		require.NotNil(t, result)
		require.Len(t, result.Content, 1, "expected one content item")

		// Parse the response
		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok, "expected text content")
		require.NotEmpty(t, textContent.Text)

		var eventResult map[string]any
		err = json.Unmarshal([]byte(textContent.Text), &eventResult)
		require.NoError(t, err, "failed to parse event result JSON")
		require.Equal(t, "event-001", eventResult["event_id"])
		require.Equal(t, "canvas-001", eventResult["canvas_id"])
		require.Equal(t, "node-1", eventResult["node_id"])
	})

	// Wait for server to finish (with timeout) when session closes
	go func() {
		select {
		case <-serverDone:
		case <-time.After(5 * time.Second):
		}
	}()
}
