package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var resetEventSourceKeyCmd = &cobra.Command{
	Use:   "event-source-key [EVENT_SOURCE_ID_OR_NAME]",
	Short: "Reset the key for a event source",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		idOrName := args[0]
		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")

		c := DefaultClient()

		response, _, err := c.EventSourceAPI.SuperplaneResetEventSourceKey(
			context.Background(),
			canvasIDOrName,
			idOrName,
		).Body(map[string]any{}).Execute()
		Check(err)

		source := response.GetEventSource()
		fmt.Printf("Key for event source %s reset successfully.\n", *source.GetMetadata().Name)
		fmt.Printf("New key: %s\n", *response.Key)
	},
}

// Root approve command
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset resources",
}

func init() {
	resetEventSourceKeyCmd.Flags().String("canvas-id", "", "Canvas ID")
	resetEventSourceKeyCmd.Flags().String("canvas-name", "", "Canvas name")

	RootCmd.AddCommand(resetCmd)
	resetCmd.AddCommand(resetEventSourceKeyCmd)
}
