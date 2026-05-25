package console

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

// addTriggerCommand wires `superplane console trigger`, exposing the same
// node trigger hook the UI invokes from node panels and table row actions.
func addTriggerCommand(root *cobra.Command, options core.BindOptions) {
	canvasID := ""
	node := ""
	hook := "run"
	parameters := ""

	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "Invoke a trigger hook on a node from the Console",
		Long: `Invoke a trigger hook on a node.

The default hook is "run" so the command mirrors the run button on
Console node panels. Provide --parameters to pass the same JSON payload
the UI would build for table row actions or for node panels with input
fields.`,
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	cmd.Flags().StringVar(&node, "node", "", "node id or name to trigger (required)")
	cmd.Flags().StringVar(&hook, "hook", "run", "trigger hook name (default: run)")
	cmd.Flags().StringVar(&parameters, "parameters", "", "JSON parameters for the trigger (use @file.json or - for stdin)")
	_ = cmd.MarkFlagRequired("node")
	core.Bind(cmd, &triggerCommand{
		canvasID:   &canvasID,
		node:       &node,
		hook:       &hook,
		parameters: &parameters,
	}, options)

	root.AddCommand(cmd)
}
