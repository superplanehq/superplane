package queue

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	var canvasID string
	var nodeID string
	var itemID string

	root := &cobra.Command{
		Use:   "queue",
		Short: "Manage canvas node queues",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List items in a node queue",
		Args:  cobra.NoArgs,
	}

	listCmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas ID")
	listCmd.Flags().StringVar(&nodeID, "node-id", "", "node ID")
	_ = listCmd.MarkFlagRequired("node-id")

	core.Bind(listCmd, &ListQueueItemsCommand{
		CanvasID: &canvasID,
		NodeID:   &nodeID,
	}, options)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an item from a node queue",
		Args:  cobra.NoArgs,
	}

	deleteCmd.Flags().StringVar(&canvasID, "canvas-id", "", "canvas ID")
	deleteCmd.Flags().StringVar(&nodeID, "node-id", "", "node ID")
	deleteCmd.Flags().StringVar(&itemID, "item-id", "", "queue item ID")
	_ = deleteCmd.MarkFlagRequired("node-id")
	_ = deleteCmd.MarkFlagRequired("item-id")
	core.Bind(deleteCmd, &DeleteQueueItemCommand{
		CanvasID: &canvasID,
		NodeID:   &nodeID,
		ItemID:   &itemID,
	}, options)

	root.AddCommand(listCmd)
	root.AddCommand(deleteCmd)

	return root
}
