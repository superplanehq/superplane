package events

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	var canvasID string
	var nodeID string
	var eventID string
	var limit int64
	var before string

	root := &cobra.Command{
		Use:     "events",
		Short:   "List canvas events and executions",
		Aliases: []string{"event"},
	}

	//
	// List command
	//
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List root events for a canvas or events for a specific node",
		Args:  cobra.NoArgs,
	}
	listCmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas ID")
	listCmd.Flags().StringVar(&nodeID, "node-id", "", "node ID")
	listCmd.Flags().Int64Var(&limit, "limit", 20, "maximum number of items to return")
	listCmd.Flags().StringVar(&before, "before", "", "return items before this timestamp (RFC3339)")
	core.Bind(listCmd, &ListEventsCommand{
		CanvasID: &canvasID,
		NodeID:   &nodeID,
		Limit:    &limit,
		Before:   &before,
	}, options)

	//
	// List executions command
	//
	listExecutionsCmd := &cobra.Command{
		Use:   "list-executions",
		Short: "List executions for a root event",
		Args:  cobra.NoArgs,
	}
	listExecutionsCmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas ID")
	listExecutionsCmd.Flags().StringVar(&eventID, "event-id", "", "event ID")
	_ = listExecutionsCmd.MarkFlagRequired("event-id")
	core.Bind(listExecutionsCmd, &ListEventExecutionsCommand{
		CanvasID: &canvasID,
		EventID:  &eventID,
	}, options)

	root.AddCommand(listCmd)
	root.AddCommand(listExecutionsCmd)

	return root
}
