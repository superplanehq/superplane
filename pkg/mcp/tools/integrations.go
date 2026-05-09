package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// RegisterIntegrationTools registers integration-related MCP tools
func RegisterIntegrationTools(ctx context.Context, s *mcp.Server, apiClient *openapi_client.APIClient) error {
	// list_integrations tool
	listIntegrationsHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListIntegrations(ctx, apiClient)
	}

	s.AddTool(&mcp.Tool{
		Name:        "list_integrations",
		Description: "List all connected integrations in the organization.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, listIntegrationsHandler)

	return nil
}

// handleListIntegrations lists all connected integrations
func handleListIntegrations(ctx context.Context, apiClient *openapi_client.APIClient) (*mcp.CallToolResult, error) {
	// First get the current user to find organization ID
	me, _, err := apiClient.MeAPI.MeMe(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if !me.User.HasOrganizationId() {
		return nil, fmt.Errorf("organization id not found for authenticated user")
	}

	orgID := me.User.GetOrganizationId()

	// Get connected integrations
	connectedResponse, _, err := apiClient.OrganizationAPI.OrganizationsListIntegrations(ctx, orgID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}

	// Get available integration definitions for labels and descriptions
	availableResponse, _, err := apiClient.IntegrationAPI.IntegrationsListIntegrations(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list available integrations: %w", err)
	}

	integrationsByName := make(map[string]openapi_client.IntegrationsIntegrationDefinition)
	for _, integration := range availableResponse.GetIntegrations() {
		integrationsByName[integration.GetName()] = integration
	}

	connected := connectedResponse.GetIntegrations()
	results := make([]map[string]any, 0, len(connected))

	for _, integration := range connected {
		metadata := integration.GetMetadata()
		status := integration.GetStatus()
		integrationName := metadata.GetIntegrationName()

		result := map[string]any{
			"id":               metadata.GetId(),
			"name":             metadata.GetName(),
			"integration_name": integrationName,
			"state":            status.GetState(),
		}

		if definition, found := integrationsByName[integrationName]; found {
			result["label"] = definition.GetLabel()
			result["description"] = definition.GetDescription()
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
