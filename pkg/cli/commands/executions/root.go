package executions

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	var canvasID string
	var nodeID string
	var executionID string
	var limit int64
	var before string

	root := &cobra.Command{
		Use:     "executions",
		Short:   "Manage canvas node executions",
		Aliases: []string{"execution"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List executions for a canvas node",
		Args:  cobra.NoArgs,
	}
	listCmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas ID")
	listCmd.Flags().StringVar(&nodeID, "node-id", "", "node ID")
	listCmd.Flags().Int64Var(&limit, "limit", 20, "maximum number of items to return")
	listCmd.Flags().StringVar(&before, "before", "", "return items before this timestamp (RFC3339)")
	_ = listCmd.MarkFlagRequired("node-id")
	core.Bind(listCmd, &ListExecutionsCommand{
		CanvasID: &canvasID,
		NodeID:   &nodeID,
		Limit:    &limit,
		Before:   &before,
	}, options)

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel an execution",
		Args:  cobra.NoArgs,
	}
	cancelCmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas ID")
	cancelCmd.Flags().StringVar(&executionID, "execution-id", "", "execution ID")
	_ = cancelCmd.MarkFlagRequired("execution-id")
	core.Bind(cancelCmd, &CancelExecutionCommand{
		CanvasID:    &canvasID,
		ExecutionID: &executionID,
	}, options)

	root.AddCommand(listCmd)
	root.AddCommand(cancelCmd)

	return root
}
