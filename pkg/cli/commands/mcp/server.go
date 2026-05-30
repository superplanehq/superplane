package mcp

import (
	"context"
	"fmt"
	"os"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// runServer builds the MCP server from the live command tree and serves it over
// stdio until the transport closes.
func runServer(cmd *cobra.Command, readOnly bool) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate the superplane binary: %w", err)
	}

	server := mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: serverName, Version: serverVersion},
		nil,
	)

	// cmd.Root() is the fully assembled CLI tree at run time, so we avoid an
	// import cycle with the parent cli package.
	for _, t := range collectTools(cmd.Root(), readOnly) {
		server.AddTool(t.tool, makeHandler(self, t.path))
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	return server.Run(ctx, &mcpsdk.StdioTransport{})
}
