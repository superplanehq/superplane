package canvases

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "canvases",
		Short:   "Manage canvases",
		Aliases: []string{"canvas"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List canvases",
		Args:  cobra.NoArgs,
	}
	core.Bind(listCmd, &listCommand{}, options)

	getCmd := &cobra.Command{
		Use:   "get <name-or-id>",
		Short: "Get a canvas",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	activeCmd := &cobra.Command{
		Use:   "active [canvas-id]",
		Short: "Set the active canvas",
		Long:  "Without arguments, prompts for a canvas selection. With a canvas ID, sets it directly.",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(activeCmd, &ActiveCommand{}, options)

	var createFile string
	createCmd := &cobra.Command{
		Use:   "create [canvas-name]",
		Short: "Create a canvas",
		Args:  cobra.MaximumNArgs(1),
	}
	createCmd.Flags().StringVarP(&createFile, "file", "f", "", "filename, directory, or URL to files to use to create the resource")
	core.Bind(createCmd, &createCommand{file: &createFile}, options)

	var updateFile string
	var updateAutoLayout string
	var updateAutoLayoutScope string
	var updateAutoLayoutNodes []string
	updateCmd := &cobra.Command{
		Use:   "update [name-or-id]",
		Short: "Update a canvas from a file",
		Args:  cobra.MaximumNArgs(1),
	}
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "filename, directory, or URL to files to use to update the resource")
	updateCmd.Flags().StringVar(&updateAutoLayout, "auto-layout", "", "automatically arrange the canvas (supported: horizontal)")
	updateCmd.Flags().StringVar(&updateAutoLayoutScope, "auto-layout-scope", "", "scope for auto layout (full-canvas, connected-component, exact-set)")
	updateCmd.Flags().StringArrayVar(&updateAutoLayoutNodes, "auto-layout-node", nil, "node id seed for auto layout (repeatable)")
	core.Bind(updateCmd, &updateCommand{
		file:            &updateFile,
		autoLayout:      &updateAutoLayout,
		autoLayoutScope: &updateAutoLayoutScope,
		autoLayoutNodes: &updateAutoLayoutNodes,
	}, options)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)
	root.AddCommand(activeCmd)
	root.AddCommand(createCmd)
	root.AddCommand(updateCmd)

	return root
}
