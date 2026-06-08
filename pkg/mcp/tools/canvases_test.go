package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const testCanvasesListResponse = `{
	"canvases": [
		{
			"metadata": {
				"id": "canvas-001",
				"name": "test-canvas",
				"createdAt": "2025-01-15T10:00:00Z"
			}
		}
	]
}`

const testCanvasDescribeResponse = `{
	"canvas": {
		"metadata": {
			"id": "canvas-001",
			"name": "test-canvas",
			"createdAt": "2025-01-15T10:00:00Z",
			"updatedAt": "2025-01-16T10:00:00Z"
		},
		"spec": {
			"nodes": [{"id": "node-1"}]
		}
	}
}`

func newTestAPIClient(t *testing.T, handler http.HandlerFunc) *openapi_client.APIClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{{URL: server.URL}}
	return openapi_client.NewAPIClient(config)
}

func TestHandleListCanvases(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/canvases", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testCanvasesListResponse))
	})

	ctx := context.Background()
	result, err := handleListCanvases(ctx, apiClient)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var canvases []map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &canvases))
	require.Len(t, canvases, 1)
	require.Equal(t, "canvas-001", canvases[0]["id"])
	require.Equal(t, "test-canvas", canvases[0]["name"])
}

func TestHandleDescribeCanvas(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/canvases/canvas-001", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testCanvasDescribeResponse))
	})

	ctx := context.Background()
	result, err := handleDescribeCanvas(ctx, apiClient, "canvas-001")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var canvas map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &canvas))
	require.Equal(t, "canvas-001", canvas["id"])
	require.Equal(t, "test-canvas", canvas["name"])
	require.Contains(t, canvas, "spec_yaml")
}
