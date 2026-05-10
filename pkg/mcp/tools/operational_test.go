package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func newOpTestClient(t *testing.T, handler http.HandlerFunc) *openapi_client.APIClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{{URL: server.URL}}
	return openapi_client.NewAPIClient(config)
}

func TestHandleDeleteCanvas(t *testing.T) {
	apiClient := newOpTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/canvases/canvas-001")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	})

	ctx := context.Background()
	result, err := handleDeleteCanvas(ctx, apiClient, "canvas-001")
	require.NoError(t, err)
	require.NotNil(t, result)

	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "deleted successfully")
}

func TestHandlePauseNode(t *testing.T) {
	apiClient := newOpTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/canvases/canvas-001/nodes/node-001/pause")

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["paused"])

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"node":{}}`)
	})

	ctx := context.Background()
	result, err := handlePauseNode(ctx, apiClient, "canvas-001", "node-001", true)
	require.NoError(t, err)

	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "paused successfully")
}

func TestHandleCancelExecution(t *testing.T) {
	apiClient := newOpTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v1/canvases/canvas-001/executions/exec-001/cancel")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	})

	ctx := context.Background()
	result, err := handleCancelExecution(ctx, apiClient, "canvas-001", "exec-001")
	require.NoError(t, err)

	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "cancelled successfully")
}

func TestHandleListTriggers(t *testing.T) {
	apiClient := newOpTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v1/triggers")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"triggers":[{"name":"webhook","label":"Webhook","description":"HTTP webhook trigger","configuration":[{"name":"authentication"}]}]}`)
	})

	ctx := context.Background()
	result, err := handleListTriggers(ctx, apiClient)
	require.NoError(t, err)

	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "webhook")
	assert.Contains(t, text, "Webhook")
	assert.Contains(t, text, "authentication")
}

func TestHandleListActions(t *testing.T) {
	apiClient := newOpTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v1/actions")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"actions":[{"name":"run-script","label":"Run Script","description":"Execute a script","configuration":[{"name":"script"}],"outputChannels":[{"name":"success"},{"name":"failure"}]}]}`)
	})

	ctx := context.Background()
	result, err := handleListActions(ctx, apiClient)
	require.NoError(t, err)

	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "run-script")
	assert.Contains(t, text, "Run Script")
	assert.Contains(t, text, "success")
	assert.Contains(t, text, "failure")
}
