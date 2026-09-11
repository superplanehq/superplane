package workspaces

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "workspace",
		Short:   "Manage workspaces",
		Aliases: []string{"workspaces"},
	}

	activeCmd := &cobra.Command{
		Use:   "active [workspace]",
		Short: "Set or show the active workspace",
		Long: `Set the active workspace used when --workspace is omitted.

Pass a workspace name, key, or UUID, or run with no args in an interactive terminal
to pick from a list. Non-interactive with no args prints the active workspace id.`,
		Args: cobra.MaximumNArgs(1),
	}
	core.Bind(activeCmd, &activeCommand{}, options)

	root.AddCommand(activeCmd)

	return root
}
