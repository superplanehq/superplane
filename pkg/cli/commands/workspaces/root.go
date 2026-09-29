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

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Long: `List workspaces in the current organization.

The KEY column is the workspace key used by --workspace and other commands.

Example:
  superplane workspaces list`,
		Args: cobra.NoArgs,
	}
	core.Bind(listCmd, &listCommand{}, options)

	var describeWorkspace string
	describeCmd := &cobra.Command{
		Use:   "describe [workspace]",
		Short: "Show workspace details",
		Long: `Show a workspace key, name, id, and lines.

Pass a workspace key or UUID as an argument or with --workspace.
When both are omitted, the active workspace from
"superplane workspace active" is used.

Examples:
  superplane workspaces describe super
  superplane workspaces describe --workspace super`,
		Args: cobra.MaximumNArgs(1),
	}
	describeCmd.Flags().StringVar(&describeWorkspace, "workspace", "", "workspace key or UUID (default: active workspace)")
	core.Bind(describeCmd, &describeCommand{
		workspace: &describeWorkspace,
	}, options)

	activeCmd := &cobra.Command{
		Use:   "active [workspace]",
		Short: "Set or show the active workspace",
		Long: `Set the active workspace used when --workspace is omitted.

Pass a workspace key or UUID, or run with no args in an interactive terminal
to pick from a list. Non-interactive with no args prints the active workspace id.`,
		Args: cobra.MaximumNArgs(1),
	}
	core.Bind(activeCmd, &activeCommand{}, options)

	root.AddCommand(listCmd)
	root.AddCommand(describeCmd)
	root.AddCommand(activeCmd)

	return root
}
