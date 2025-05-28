package cli

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

var approveEventCmd = &cobra.Command{
	Use:     "event [EVENT_ID]",
	Short:   "Approve a stage event",
	Long:    `Approve a pending stage event that requires approval.`,
	Aliases: []string{"events"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		eventID := args[0]

		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")
		stageIDOrName := getOneOrAnotherFlag(cmd, "stage-id", "stage-name")

		c := DefaultClient()

		request := openapi_client.NewSuperplaneApproveStageEventBody()
		request.SetRequesterId(uuid.NewString())

		response, _, err := c.EventAPI.SuperplaneApproveStageEvent(
			context.Background(),
			canvasIDOrName,
			stageIDOrName,
			eventID,
		).Body(*request).Execute()
		Check(err)

		fmt.Printf("Event '%s' approved successfully.\n", *response.Event.Id)
	},
}

// Root approve command
var approveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Approve resources that need approval",
	Long:  `Approve events or other resources that need approval.`,
}

func init() {
	approveEventCmd.Flags().String("canvas-id", "", "Canvas ID")
	approveEventCmd.Flags().String("canvas-name", "", "Canvas name")
	approveEventCmd.Flags().String("stage-id", "", "Stage ID")
	approveEventCmd.Flags().String("stage-name", "", "Stage name")

	RootCmd.AddCommand(approveCmd)
	approveCmd.AddCommand(approveEventCmd)
}
