package console

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

// addDataCommand wires `superplane console data`. The implementation is in
// data_runtime.go to keep the data-source resolution logic separate from
// the Cobra wiring.
func addDataCommand(root *cobra.Command, options core.BindOptions) {
	canvasID := ""
	limit := int64(0)
	cmd := &cobra.Command{
		Use:   "data <panel-id>",
		Short: "Fetch the runtime data backing a Console panel",
		Long: `Fetch the runtime data backing a Console panel.

This command reads the panel's data source and returns the rows or
computed value the UI would render. Supported sources are memory,
executions, and runs (matching the panel types defined by the API).
Markdown and node panels do not have a data source and produce a
descriptive error instead.`,
		Args: cobra.ExactArgs(1),
	}
	cmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas id (defaults to the active canvas)")
	cmd.Flags().Int64Var(&limit, "limit", 0, "override the panel data source limit (0 keeps the panel's configured limit)")
	core.Bind(cmd, &dataCommand{canvasID: &canvasID, limit: &limit}, options)

	root.AddCommand(cmd)
}
