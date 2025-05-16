package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/superplanehq/superplane/pkg/cli/utils"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

var approveEventCmd = &cobra.Command{
	Use:     "event [CANVAS_ID] [STAGE_ID] [EVENT_ID]",
	Short:   "Approve a stage event",
	Long:    `Approve a pending stage event that requires approval.`,
	Aliases: []string{"events"},
	Args:    cobra.ExactArgs(3),

	Run: func(cmd *cobra.Command, args []string) {
		canvasID := args[0]
		stageID := args[1]
		eventID := args[2]
		requesterID, _ := cmd.Flags().GetString("requester-id")

		c := DefaultClient()

		request := openapi_client.NewSuperplaneApproveStageEventBody()
		request.SetRequesterId(requesterID)

		response, _, err := c.EventAPI.SuperplaneApproveStageEvent(
			context.Background(),
			canvasID,
			stageID,
			eventID,
		).Body(*request).Execute()
		utils.Check(err)

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
	RootCmd.AddCommand(approveCmd)
	approveCmd.AddCommand(approveEventCmd)
	approveEventCmd.Flags().String("requester-id", "", "ID of the user approving the event")
}
