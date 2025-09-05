package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var cancelEventCmd = &cobra.Command{
	Use:     "event [EVENT_ID]",
	Short:   "Cancel a stage event",
	Long:    `Cancel a pending stage event that is waiting for execution.`,
	Aliases: []string{"events"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		eventID := args[0]

		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name", true)
		stageIDOrName := getOneOrAnotherFlag(cmd, "stage-id", "stage-name", true)

		c := DefaultClient()

		response, _, err := c.StageAPI.SuperplaneCancelStageEvent(context.Background(), canvasIDOrName, stageIDOrName, eventID).
			Body(map[string]any{}).
			Execute()

		Check(err)

		fmt.Printf("Event '%s' cancelled successfully.\n", *response.Event.Id)
	},
}

// Root cancel command
var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel resources that are pending execution",
	Long:  `Cancel events or other resources that are pending execution.`,
}

func init() {
	cancelEventCmd.Flags().String("canvas-id", "", "Canvas ID")
	cancelEventCmd.Flags().String("canvas-name", "", "Canvas name")
	cancelEventCmd.Flags().String("stage-id", "", "Stage ID")
	cancelEventCmd.Flags().String("stage-name", "", "Stage name")

	RootCmd.AddCommand(cancelCmd)
	cancelCmd.AddCommand(cancelEventCmd)
}