package files

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewRootCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "files",
		Short: "Read canvas repository files",
		Long:  "List and read files stored in the git repository attached to a canvas.",
	}

	treeCmd := &cobra.Command{
		Use:   "tree [canvas-name-or-id]",
		Short: "List canvas repository files as a tree",
		Long: `List files in the canvas git repository as a directory tree.

The canvas argument is optional. When omitted, the active canvas
configured with "superplane canvases active" is used.`,
		Args: cobra.MaximumNArgs(1),
	}

	core.Bind(treeCmd, &TreeCommand{}, options)

	showCmd := &cobra.Command{
		Use:   "show <path> [canvas-name-or-id]",
		Short: "Show a canvas repository file",
		Long: `Print the contents of a file from the canvas git repository.

The canvas argument is optional. When omitted, the active canvas
configured with "superplane canvases active" is used.`,
		Args: cobra.RangeArgs(1, 2),
	}

	core.Bind(showCmd, &ShowCommand{}, options)

	root.AddCommand(treeCmd)
	root.AddCommand(showCmd)

	return root
}
