package console

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

// NewCommand wires the `superplane console` command tree.
//
// User-facing terminology stays "Console" even though the underlying API
// still calls this resource "Dashboard" (see canvas_dashboard.proto).
// Updating the user-visible text here without updating the backend keeps
// the migration story consistent: API methods follow the legacy name, and
// only display strings change.
func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "console",
		Short: "Manage canvas Console (panels, layouts, and runtime data)",
		Long: `Manage the canvas Console.

Every Console subcommand resolves the target canvas with --canvas-id (or
the active canvas configured via "superplane canvases active"). Imports
replace the entire Console for the canvas to match the API behavior.`,
		Aliases: []string{"dashboard"},
	}

	addConsoleCommands(root, options)
	addPanelCommands(root, options)
	addRuntimeCommands(root, options)

	return root
}

func addConsoleCommands(root *cobra.Command, options core.BindOptions) {
	getCanvasID := ""
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Show a summary of the canvas Console",
		Args:  cobra.NoArgs,
	}
	getCmd.Flags().StringVar(&getCanvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	core.Bind(getCmd, &getCommand{canvasID: &getCanvasID}, options)

	exportCanvasID := ""
	exportFile := ""
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export a canvas Console as YAML",
		Args:  cobra.NoArgs,
	}
	exportCmd.Flags().StringVar(&exportCanvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	exportCmd.Flags().StringVarP(&exportFile, "file", "f", "", "output file path (use - or omit to write to stdout)")
	core.Bind(exportCmd, &exportCommand{canvasID: &exportCanvasID, file: &exportFile}, options)

	importCanvasID := ""
	importFile := ""
	importYes := false
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Replace a canvas Console with the contents of a YAML file",
		Args:  cobra.NoArgs,
	}
	importCmd.Flags().StringVar(&importCanvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	importCmd.Flags().StringVarP(&importFile, "file", "f", "", "console YAML file (use - to read from stdin)")
	importCmd.Flags().BoolVarP(&importYes, "yes", "y", false, "do not prompt for replace-all confirmation")
	_ = importCmd.MarkFlagRequired("file")
	core.Bind(importCmd, &importCommand{canvasID: &importCanvasID, file: &importFile, yes: &importYes}, options)

	clearCanvasID := ""
	clearYes := false
	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove all panels and layout from a canvas Console",
		Args:  cobra.NoArgs,
	}
	clearCmd.Flags().StringVar(&clearCanvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	clearCmd.Flags().BoolVarP(&clearYes, "yes", "y", false, "do not prompt for confirmation")
	core.Bind(clearCmd, &clearCommand{canvasID: &clearCanvasID, yes: &clearYes}, options)

	root.AddCommand(getCmd)
	root.AddCommand(exportCmd)
	root.AddCommand(importCmd)
	root.AddCommand(clearCmd)
}

func addPanelCommands(root *cobra.Command, options core.BindOptions) {
	panels := &cobra.Command{
		Use:   "panels",
		Short: "Manage individual Console panels",
	}

	listCanvasID := ""
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List Console panels",
		Args:  cobra.NoArgs,
	}
	listCmd.Flags().StringVar(&listCanvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	core.Bind(listCmd, &panelsListCommand{canvasID: &listCanvasID}, options)

	getCanvasID := ""
	getCmd := &cobra.Command{
		Use:   "get <panel-id>",
		Short: "Show a Console panel",
		Args:  cobra.ExactArgs(1),
	}
	getCmd.Flags().StringVar(&getCanvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	core.Bind(getCmd, &panelsGetCommand{canvasID: &getCanvasID}, options)

	deleteCanvasID := ""
	deleteYes := false
	deleteCmd := &cobra.Command{
		Use:   "delete <panel-id>",
		Short: "Delete a Console panel",
		Args:  cobra.ExactArgs(1),
	}
	deleteCmd.Flags().StringVar(&deleteCanvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "do not prompt for confirmation")
	core.Bind(deleteCmd, &panelsDeleteCommand{canvasID: &deleteCanvasID, yes: &deleteYes}, options)

	upsertCanvasID := ""
	upsertFile := ""
	upsertLayout := ""
	upsertCmd := &cobra.Command{
		Use:   "upsert",
		Short: "Add or update a Console panel from a YAML/JSON file",
		Long: `Add or update a Console panel.

The file describes a single panel and may include a layout block. Use
` + "`--layout '{\"x\":0,\"y\":0,\"w\":4,\"h\":3}'`" + ` to override the layout
position from the command line. When the layout is not provided, the
existing layout entry for the panel is preserved (or left for the API to
default for new panels).`,
		Args: cobra.NoArgs,
	}
	upsertCmd.Flags().StringVar(&upsertCanvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	upsertCmd.Flags().StringVarP(&upsertFile, "file", "f", "", "panel definition file (use - for stdin)")
	upsertCmd.Flags().StringVar(&upsertLayout, "layout", "", "JSON object overriding the panel layout")
	_ = upsertCmd.MarkFlagRequired("file")
	core.Bind(upsertCmd, &panelsUpsertCommand{canvasID: &upsertCanvasID, file: &upsertFile, layout: &upsertLayout}, options)

	panels.AddCommand(listCmd)
	panels.AddCommand(getCmd)
	panels.AddCommand(deleteCmd)
	panels.AddCommand(upsertCmd)

	root.AddCommand(panels)
}
