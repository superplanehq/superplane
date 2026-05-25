package widgets

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

// NewCommand wires the `superplane widgets` command tree.
//
// These commands manage canvas widget *instances* — TYPE_WIDGET nodes
// embedded in a canvas spec (annotations, etc.). To discover the widget
// definitions registered with the platform see `superplane index widgets`.
func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "widgets",
		Short: "Manage widget nodes embedded in a canvas",
		Long: `Manage widget nodes embedded in a canvas (TYPE_WIDGET).

These commands operate on widget instances inside a canvas spec; for the
catalog of available widgets see "superplane index widgets". Mutations
go through the regular canvas draft flow, so --draft is required when
the canvas has change management enabled.`,
	}

	addListCommand(root, options)
	addGetCommand(root, options)
	addAddCommand(root, options)
	addUpdateCommand(root, options)
	addDeleteCommand(root, options)

	return root
}

func addListCommand(root *cobra.Command, options core.BindOptions) {
	canvasID := ""
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List widget nodes on a canvas",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	core.Bind(cmd, &listCommand{canvasID: &canvasID}, options)
	root.AddCommand(cmd)
}

func addGetCommand(root *cobra.Command, options core.BindOptions) {
	canvasID := ""
	cmd := &cobra.Command{
		Use:   "get <widget-id-or-name>",
		Short: "Show a widget node",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	core.Bind(cmd, &getCommand{canvasID: &canvasID}, options)
	root.AddCommand(cmd)
}

func addAddCommand(root *cobra.Command, options core.BindOptions) {
	canvasID := ""
	component := ""
	name := ""
	configuration := ""
	positionX := int32(0)
	positionY := int32(0)
	width := int32(0)
	height := int32(0)
	color := ""
	text := ""
	draft := false

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a widget node to a canvas",
		Long: `Add a widget node to a canvas.

Use --component to pick the widget definition (see "superplane index
widgets"), and --configuration with inline JSON, @file.json, or - for
stdin. Annotation widgets accept --text, --color, --width, and --height
shortcuts that are merged into the configuration when not already set.`,
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	cmd.Flags().StringVar(&component, "component", "", "widget component name (e.g. annotation)")
	cmd.Flags().StringVar(&name, "name", "", "node name (defaults to the component)")
	cmd.Flags().StringVar(&configuration, "configuration", "", "JSON configuration (inline, @file, or -)")
	cmd.Flags().Int32Var(&positionX, "position-x", 0, "node x position")
	cmd.Flags().Int32Var(&positionY, "position-y", 0, "node y position")
	cmd.Flags().Int32Var(&width, "width", 0, "annotation width (annotation-friendly shortcut)")
	cmd.Flags().Int32Var(&height, "height", 0, "annotation height (annotation-friendly shortcut)")
	cmd.Flags().StringVar(&color, "color", "", "annotation color (annotation-friendly shortcut)")
	cmd.Flags().StringVar(&text, "text", "", "annotation text (annotation-friendly shortcut)")
	cmd.Flags().BoolVar(&draft, "draft", false, "keep the change as a draft instead of auto-publishing")
	_ = cmd.MarkFlagRequired("component")

	command := &addCommand{
		canvasID:      &canvasID,
		component:     &component,
		name:          &name,
		configuration: &configuration,
		positionX:     &positionX,
		positionY:     &positionY,
		width:         &width,
		height:        &height,
		color:         &color,
		text:          &text,
		draft:         &draft,
	}
	core.Bind(cmd, command, options)
	root.AddCommand(cmd)
}

func addUpdateCommand(root *cobra.Command, options core.BindOptions) {
	canvasID := ""
	configuration := ""
	name := ""
	positionX := int32(0)
	positionY := int32(0)
	width := int32(0)
	height := int32(0)
	color := ""
	text := ""
	draft := false

	cmd := &cobra.Command{
		Use:   "update <widget-id-or-name>",
		Short: "Update a widget node",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	cmd.Flags().StringVar(&configuration, "configuration", "", "JSON configuration override (inline, @file, or -)")
	cmd.Flags().StringVar(&name, "name", "", "rename the node")
	cmd.Flags().Int32Var(&positionX, "position-x", 0, "set node x position")
	cmd.Flags().Int32Var(&positionY, "position-y", 0, "set node y position")
	cmd.Flags().Int32Var(&width, "width", 0, "annotation width (annotation-friendly shortcut)")
	cmd.Flags().Int32Var(&height, "height", 0, "annotation height (annotation-friendly shortcut)")
	cmd.Flags().StringVar(&color, "color", "", "annotation color (annotation-friendly shortcut)")
	cmd.Flags().StringVar(&text, "text", "", "annotation text (annotation-friendly shortcut)")
	cmd.Flags().BoolVar(&draft, "draft", false, "keep the change as a draft instead of auto-publishing")

	command := &updateCommand{
		canvasID:      &canvasID,
		configuration: &configuration,
		name:          &name,
		positionX:     &positionX,
		positionY:     &positionY,
		width:         &width,
		height:        &height,
		color:         &color,
		text:          &text,
		draft:         &draft,
	}
	core.Bind(cmd, command, options)
	root.AddCommand(cmd)
}

func addDeleteCommand(root *cobra.Command, options core.BindOptions) {
	canvasID := ""
	yes := false
	draft := false

	cmd := &cobra.Command{
		Use:   "delete <widget-id-or-name>",
		Short: "Delete a widget node",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not prompt for confirmation")
	cmd.Flags().BoolVar(&draft, "draft", false, "keep the change as a draft instead of auto-publishing")
	core.Bind(cmd, &deleteCommand{canvasID: &canvasID, yes: &yes, draft: &draft}, options)
	root.AddCommand(cmd)
}
