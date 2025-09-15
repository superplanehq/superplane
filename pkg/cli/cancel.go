package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var discardEventCmd = &cobra.Command{
	Use:     "event [EVENT_ID]",
	Short:   "Discard a stage event",
	Long:    `Discards a stage event from the queue`,
	Aliases: []string{"events"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		eventID := args[0]

		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name", true)
		stageIDOrName := getOneOrAnotherFlag(cmd, "stage-id", "stage-name", true)

		c := DefaultClient()

		response, _, err := c.StageAPI.SuperplaneDiscardStageEvent(context.Background(), canvasIDOrName, stageIDOrName, eventID).
			Body(map[string]any{}).
			Execute()

		Check(err)

		fmt.Printf("Event '%s' discarded successfully.\n", *response.Event.Id)
	},
}

var discardCmd = &cobra.Command{
	Use:   "discard",
	Short: "Discard resources",
}

func init() {
	discardEventCmd.Flags().String("canvas-id", "", "Canvas ID")
	discardEventCmd.Flags().String("canvas-name", "", "Canvas name")
	discardEventCmd.Flags().String("stage-id", "", "Stage ID")
	discardEventCmd.Flags().String("stage-name", "", "Stage name")

	RootCmd.AddCommand(discardCmd)
	discardCmd.AddCommand(discardEventCmd)
}
