package runs

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	var canvasID string
	var nodeID string
	var runID string
	var limit int64
	var before string

	root := &cobra.Command{
		Use:     "runs",
		Short:   "Manage canvas node runs",
		Aliases: []string{"run"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List runs for a canvas node",
		Args:  cobra.NoArgs,
	}
	listCmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas ID")
	listCmd.Flags().StringVar(&nodeID, "node-id", "", "node ID")
	listCmd.Flags().Int64Var(&limit, "limit", 20, "maximum number of items to return")
	listCmd.Flags().StringVar(&before, "before", "", "return items before this timestamp (RFC3339)")
	_ = listCmd.MarkFlagRequired("node-id")
	core.Bind(listCmd, &ListRunsCommand{
		CanvasID: &canvasID,
		NodeID:   &nodeID,
		Limit:    &limit,
		Before:   &before,
	}, options)

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a run",
		Args:  cobra.NoArgs,
	}
	cancelCmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas ID")
	cancelCmd.Flags().StringVar(&runID, "run-id", "", "run ID")
	_ = cancelCmd.MarkFlagRequired("run-id")
	core.Bind(cancelCmd, &CancelRunCommand{
		CanvasID: &canvasID,
		RunID:    &runID,
	}, options)

	root.AddCommand(listCmd)
	root.AddCommand(cancelCmd)

	return root
}
