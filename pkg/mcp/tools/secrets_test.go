package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

const testSecretsListResponse = `{
	"secrets": [
		{
			"metadata": {
				"id": "secret-001",
				"name": "my-secret",
				"createdAt": "2025-01-15T10:00:00Z"
			}
		}
	]
}`

func TestHandleListSecrets(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/me":
			_, _ = w.Write([]byte(testMeResponse))
		case "/api/v1/secrets":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "DOMAIN_TYPE_ORGANIZATION", r.URL.Query().Get("domainType"))
			require.Equal(t, "org-001", r.URL.Query().Get("domainId"))
			_, _ = w.Write([]byte(testSecretsListResponse))
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	})

	ctx := context.Background()
	result, err := handleListSecrets(ctx, apiClient)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var secrets []map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &secrets))
	require.Len(t, secrets, 1)
	require.Equal(t, "secret-001", secrets[0]["id"])
	require.Equal(t, "my-secret", secrets[0]["name"])
	require.Contains(t, secrets[0], "created_at")
}
