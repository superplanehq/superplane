package mcp

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

const (
	serverName    = "superplane"
	serverVersion = "0.1.0"
)

// NewCommand returns the `superplane mcp` command. It runs a Model Context
// Protocol (MCP) server over stdio that auto-exposes every SuperPlane CLI
// command as an MCP tool, so AI coding agents (Claude Code, Codex) can drive
// SuperPlane directly.
//
// The options argument is accepted for symmetry with the other command groups
// but is unused: tools execute by shelling out to this same binary, reusing the
// CLI's existing context/auth on disk.
func NewCommand(_ core.BindOptions) *cobra.Command {
	var readOnly bool

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run an MCP server exposing the SuperPlane CLI to AI coding agents",
		Long: "Starts a Model Context Protocol (MCP) server over stdio.\n\n" +
			"Every SuperPlane CLI command is auto-exposed as an MCP tool, so agents like\n" +
			"Claude Code and Codex can list canvases, create workflows, inspect executions,\n" +
			"and more by calling tools instead of memorizing the CLI.\n\n" +
			"Add it to Claude Code with:\n" +
			"  claude mcp add superplane -- superplane mcp",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd, readOnly)
		},
	}

	cmd.Flags().BoolVar(&readOnly, "read-only", false,
		"expose only read-only tools (list/get/describe/show/...)")

	return cmd
}
