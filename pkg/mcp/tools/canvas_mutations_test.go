package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const testCreateCanvasYAML = `apiVersion: v1
kind: Canvas
metadata:
  name: test-canvas
spec:
  nodes: []
  edges: []`

const testCreateCanvasResponse = `{
	"canvas": {
		"metadata": {
			"id": "canvas-123",
			"name": "test-canvas",
			"createdAt": "2025-01-15T10:00:00Z"
		},
		"spec": {
			"nodes": [],
			"edges": []
		}
	}
}`

const testListVersionsResponse = `{
	"versions": [
		{
			"metadata": {
				"id": "version-001",
				"canvasId": "canvas-123",
				"state": "STATE_DRAFT"
			},
			"spec": {
				"nodes": [],
				"edges": []
			}
		}
	]
}`

const testUpdateCanvasResponse = `{
	"version": {
		"metadata": {
			"id": "version-001",
			"canvasId": "canvas-123",
			"state": "STATE_DRAFT"
		},
		"spec": {
			"nodes": [],
			"edges": []
		}
	}
}`

const testPublishCanvasResponse = `{
	"version": {
		"metadata": {
			"id": "version-001",
			"canvasId": "canvas-123",
			"state": "STATE_PUBLISHED"
		}
	}
}`

const testDescribeVersionResponse = `{
	"version": {
		"metadata": {
			"id": "version-001",
			"canvasId": "canvas-123",
			"state": "STATE_PUBLISHED"
		},
		"spec": {
			"nodes": [],
			"edges": []
		}
	}
}`

func TestHandleCreateCanvas(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle the versions list call (for validation check)
		if strings.Contains(r.URL.Path, "/versions") && r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"v1","canvasId":"canvas-123","state":"STATE_PUBLISHED"},"spec":{"nodes":[],"edges":[]}}]}`))
			return
		}

		require.Equal(t, "/api/v1/canvases", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		// Verify request body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		require.Contains(t, req, "canvas")

		_, _ = w.Write([]byte(testCreateCanvasResponse))
	})

	ctx := context.Background()
	result, err := handleCreateCanvas(ctx, apiClient, "test-canvas", testCreateCanvasYAML)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var response map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &response))
	require.Equal(t, "canvas-123", response["id"])
	require.Equal(t, "test-canvas", response["name"])
	require.Contains(t, response, "message")
}

func TestHandleUpdateCanvas(t *testing.T) {
	callCount := 0
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/versions") && r.Method == http.MethodGet:
			// List versions
			_, _ = w.Write([]byte(testListVersionsResponse))
		case strings.HasSuffix(r.URL.Path, "/versions") && r.Method == http.MethodPut:
			// Update version (PUT to /canvases/{id}/versions)
			_, _ = w.Write([]byte(testUpdateCanvasResponse))
		case strings.Contains(r.URL.Path, "/publish") && r.Method == http.MethodPatch:
			// Publish version
			_, _ = w.Write([]byte(testPublishCanvasResponse))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	ctx := context.Background()
	result, err := handleUpdateCanvas(ctx, apiClient, "canvas-123", testCreateCanvasYAML)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var response map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &response))
	require.Equal(t, "canvas-123", response["canvas_id"])
	require.Equal(t, "version-001", response["version_id"])
	require.Equal(t, "published", response["state"])
}

func TestHandlePublishCanvasVersion(t *testing.T) {
	callCount := 0
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/publish") {
			// Publish
			_, _ = w.Write([]byte(testPublishCanvasResponse))
		} else {
			// Describe
			_, _ = w.Write([]byte(testDescribeVersionResponse))
		}
	})

	ctx := context.Background()
	result, err := handlePublishCanvasVersion(ctx, apiClient, "canvas-123", "version-001")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var response map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &response))
	require.Equal(t, "canvas-123", response["canvas_id"])
	require.Equal(t, "version-001", response["version_id"])
}

func TestHandleValidateCanvas(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.Path, "/versions/")
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testDescribeVersionResponse))
	})

	ctx := context.Background()
	result, err := handleValidateCanvas(ctx, apiClient, "canvas-123", "version-001")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var response map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &response))
	require.Equal(t, "canvas-123", response["canvas_id"])
	require.Equal(t, "version-001", response["version_id"])
	require.Equal(t, true, response["valid"])
}

func TestFormatNodeErrors(t *testing.T) {
	tests := []struct {
		name     string
		version  openapi_client.CanvasesCanvasVersion
		expected string
	}{
		{
			name: "no errors",
			version: openapi_client.CanvasesCanvasVersion{
				Spec: &openapi_client.CanvasesCanvasSpec{
					Nodes: []openapi_client.SuperplaneComponentsNode{
						{
							Id:   ptrString("node1"),
							Name: ptrString("Test Node"),
						},
					},
				},
			},
			expected: "",
		},
		{
			name: "with error",
			version: openapi_client.CanvasesCanvasVersion{
				Spec: &openapi_client.CanvasesCanvasSpec{
					Nodes: []openapi_client.SuperplaneComponentsNode{
						{
							Id:           ptrString("node1"),
							Name:         ptrString("Test Node"),
							ErrorMessage: ptrString("Invalid configuration"),
						},
					},
				},
			},
			expected: "node node1 (Test Node): Invalid configuration",
		},
		{
			name: "multiple errors",
			version: openapi_client.CanvasesCanvasVersion{
				Spec: &openapi_client.CanvasesCanvasSpec{
					Nodes: []openapi_client.SuperplaneComponentsNode{
						{
							Id:           ptrString("node1"),
							Name:         ptrString("Node 1"),
							ErrorMessage: ptrString("Error 1"),
						},
						{
							Id:           ptrString("node2"),
							Name:         ptrString("Node 2"),
							ErrorMessage: ptrString("Error 2"),
						},
					},
				},
			},
			expected: "node node1 (Node 1): Error 1\nnode node2 (Node 2): Error 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNodeErrors(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatNodeWarnings(t *testing.T) {
	tests := []struct {
		name     string
		version  openapi_client.CanvasesCanvasVersion
		expected string
	}{
		{
			name: "no warnings",
			version: openapi_client.CanvasesCanvasVersion{
				Spec: &openapi_client.CanvasesCanvasSpec{
					Nodes: []openapi_client.SuperplaneComponentsNode{
						{
							Id:   ptrString("node1"),
							Name: ptrString("Test Node"),
						},
					},
				},
			},
			expected: "",
		},
		{
			name: "with warning",
			version: openapi_client.CanvasesCanvasVersion{
				Spec: &openapi_client.CanvasesCanvasSpec{
					Nodes: []openapi_client.SuperplaneComponentsNode{
						{
							Id:             ptrString("node1"),
							Name:           ptrString("Test Node"),
							WarningMessage: ptrString("Deprecated field used"),
						},
					},
				},
			},
			expected: "node node1 (Test Node): Deprecated field used",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNodeWarnings(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func ptrString(s string) *string {
	return &s
}
