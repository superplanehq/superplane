package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

const testMeResponse = `{
	"user": {
		"id": "user-001",
		"organizationId": "org-001"
	}
}`

const testIntegrationsListResponse = `{
	"integrations": [
		{
			"metadata": {
				"id": "int-001",
				"name": "my-integration",
				"integrationName": "github"
			},
			"status": {
				"state": "CONNECTED"
			}
		}
	]
}`

const testAvailableIntegrationsResponse = `{
	"integrations": [
		{
			"name": "github",
			"label": "GitHub",
			"description": "GitHub integration"
		}
	]
}`

func TestHandleListIntegrations(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/me":
			_, _ = w.Write([]byte(testMeResponse))
		case "/api/v1/organizations/org-001/integrations":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(testIntegrationsListResponse))
		case "/api/v1/integrations":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(testAvailableIntegrationsResponse))
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	})

	ctx := context.Background()
	result, err := handleListIntegrations(ctx, apiClient)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var integrations []map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &integrations))
	require.Len(t, integrations, 1)
	require.Equal(t, "int-001", integrations[0]["id"])
	require.Equal(t, "my-integration", integrations[0]["name"])
	require.Equal(t, "github", integrations[0]["integration_name"])
	require.Equal(t, "CONNECTED", integrations[0]["state"])
	require.Equal(t, "GitHub", integrations[0]["label"])
	require.Equal(t, "GitHub integration", integrations[0]["description"])
}
