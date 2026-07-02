package files

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewRootCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "files",
		Short: "Read app repository files",
		Long:  "List and read files stored in the git repository attached to an app.",
	}

	treeCmd := &cobra.Command{
		Use:   "tree [app-name-or-id]",
		Short: "List app repository files as a tree",
		Long: `List files in the app git repository as a directory tree.

The app argument is optional. When omitted, the active app
configured with "superplane apps active" is used.`,
		Args: cobra.MaximumNArgs(1),
	}

	core.Bind(treeCmd, &TreeCommand{}, options)

	showCmd := &cobra.Command{
		Use:   "show <path> [app-name-or-id]",
		Short: "Show an app repository file",
		Long: `Print the contents of a file from the app git repository.

The app argument is optional. When omitted, the active app
configured with "superplane apps active" is used.`,
		Args: cobra.RangeArgs(1, 2),
	}

	core.Bind(showCmd, &ShowCommand{}, options)

	root.AddCommand(treeCmd)
	root.AddCommand(showCmd)

	return root
}
