package mcp

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/mcp"
)

// NewCommand creates the mcp command
func NewCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the SuperPlane MCP server",
		Long:  "Start the Model Context Protocol (MCP) server for SuperPlane. The server runs on stdio transport and provides read-only tools for canvases, integrations, and secrets.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcp.StartServer(version)
		},
	}

	return cmd
}
