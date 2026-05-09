package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// RegisterSecretTools registers secret-related MCP tools
func RegisterSecretTools(ctx context.Context, s *mcp.Server, apiClient *openapi_client.APIClient) error {
	// list_secrets tool
	listSecretsHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListSecrets(ctx, apiClient)
	}

	s.AddTool(&mcp.Tool{
		Name:        "list_secrets",
		Description: "List all available secrets in the organization. Returns secret names only, not values.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, listSecretsHandler)

	return nil
}

// handleListSecrets lists all secrets (names only, no values)
func handleListSecrets(ctx context.Context, apiClient *openapi_client.APIClient) (*mcp.CallToolResult, error) {
	// First get the current user to find organization ID
	me, _, err := apiClient.MeAPI.MeMe(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if !me.User.HasOrganizationId() {
		return nil, fmt.Errorf("organization id not found for authenticated user")
	}

	orgID := me.User.GetOrganizationId()

	// Get secrets
	response, _, err := apiClient.SecretAPI.
		SecretsListSecrets(ctx).
		DomainType(string(openapi_client.AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_ORGANIZATION)).
		DomainId(orgID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	secrets := response.GetSecrets()
	results := make([]map[string]any, 0, len(secrets))

	for _, secret := range secrets {
		metadata := secret.GetMetadata()

		result := map[string]any{
			"id":   metadata.GetId(),
			"name": metadata.GetName(),
		}

		if metadata.HasCreatedAt() {
			result["created_at"] = metadata.GetCreatedAt()
		}

		results = append(results, result)
	}

	content, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
	}, nil
}
