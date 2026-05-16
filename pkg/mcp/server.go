package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/superplanehq/superplane/pkg/mcp/tools"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// StartServer initializes and starts the MCP server
func StartServer(version string) error {
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	apiClient := NewAPIClient(config)

	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "superplane",
			Version: version,
		},
		nil,
	)

	// Register tools
	if err := registerTools(context.Background(), s, apiClient); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	// Connect using stdio transport
	ctx := context.Background()
	transport := &mcp.StdioTransport{}
	_, err = s.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Block forever - the server will handle requests on stdio
	select {}
}

// registerTools registers all MCP tools
func registerTools(ctx context.Context, s *mcp.Server, apiClient *openapi_client.APIClient) error {
	// Canvas tools
	if err := tools.RegisterCanvasTools(ctx, s, apiClient); err != nil {
		return fmt.Errorf("failed to register canvas tools: %w", err)
	}

	// Integration tools
	if err := tools.RegisterIntegrationTools(ctx, s, apiClient); err != nil {
		return fmt.Errorf("failed to register integration tools: %w", err)
	}

	// Secret tools
	if err := tools.RegisterSecretTools(ctx, s, apiClient); err != nil {
		return fmt.Errorf("failed to register secret tools: %w", err)
	}

	return nil
}
