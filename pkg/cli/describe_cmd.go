package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/superplanehq/superplane/pkg/cli/utils"
)

var describeCanvasCmd = &cobra.Command{
	Use:     "canvas [ID]",
	Short:   "Get canvas details",
	Long:    `Retrieve details about a specific canvas`,
	Aliases: []string{"canvases"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		name, _ := cmd.Flags().GetString("name")

		c := DefaultClient()
		response, _, err := c.CanvasAPI.SuperplaneDescribeCanvas(context.Background(), id).Name(name).Execute()
		utils.Check(err)

		fmt.Printf("Canvas '%s' (ID: %s)\n", *response.Canvas.Name, *response.Canvas.Id)
		fmt.Printf("Created by: %s\n", *response.Canvas.CreatedBy)
		fmt.Printf("Created at: %s\n", *response.Canvas.CreatedAt)
	},
}

var describeEventSourceCmd = &cobra.Command{
	Use:     "event-source [CANVAS_ID] [ID]",
	Short:   "Get event source details",
	Long:    `Retrieve details about a specific event source`,
	Aliases: []string{"event-sources", "eventsource", "eventsources"},
	Args:    cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {
		canvasID := args[0]
		id := args[1]
		name, _ := cmd.Flags().GetString("name")

		c := DefaultClient()
		response, _, err := c.EventSourceAPI.SuperplaneDescribeEventSource(
			context.Background(),
			canvasID,
			id,
		).Name(name).Execute()
		utils.Check(err)

		fmt.Printf("Event Source '%s' (ID: %s)\n", *response.EventSource.Name, *response.EventSource.Id)
		fmt.Printf("Canvas: %s\n", *response.EventSource.CanvasId)
		fmt.Printf("Created at: %s\n", *response.EventSource.CreatedAt)
	},
}

var describeStageCmd = &cobra.Command{
	Use:     "stage [CANVAS_ID] [ID]",
	Short:   "Get stage details",
	Long:    `Retrieve details about a specific stage`,
	Aliases: []string{"stages"},
	Args:    cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {
		canvasID := args[0]
		id := args[1]
		name, _ := cmd.Flags().GetString("name")

		c := DefaultClient()
		response, _, err := c.StageAPI.SuperplaneDescribeStage(
			context.Background(),
			canvasID,
			id,
		).Name(name).Execute()
		utils.Check(err)

		stage := response.Stage
		fmt.Printf("Stage '%s' (ID: %s)\n", *stage.Name, *stage.Id)
		fmt.Printf("Canvas: %s\n", *stage.CanvasId)
		fmt.Printf("Created at: %s\n", *stage.CreatedAt)

		if len(stage.Connections) > 0 {
			fmt.Println("\nConnections:")
			for i, conn := range stage.Connections {
				fmt.Printf("  %d. %s (%s)\n", i+1, *conn.Name, conn.GetType())
			}
		}

		if len(stage.Conditions) > 0 {
			fmt.Println("\nConditions:")
			for i, cond := range stage.Conditions {
				fmt.Printf("  %d. %s\n", i+1, cond.GetType())
			}
		}

		if stage.RunTemplate != nil && stage.RunTemplate.GetType() == "TYPE_SEMAPHORE" {
			fmt.Println("\nRun Template:")
			fmt.Printf("  Type: Semaphore\n")
			if stage.RunTemplate.Semaphore != nil {
				if stage.RunTemplate.Semaphore.ProjectId != nil {
					fmt.Printf("  Project ID: %s\n", *stage.RunTemplate.Semaphore.ProjectId)
				}
				if stage.RunTemplate.Semaphore.Branch != nil {
					fmt.Printf("  Branch: %s\n", *stage.RunTemplate.Semaphore.Branch)
				}
				if stage.RunTemplate.Semaphore.PipelineFile != nil {
					fmt.Printf("  Pipeline File: %s\n", *stage.RunTemplate.Semaphore.PipelineFile)
				}
			}
		}
	},
}

// Root describe command
var describeCmd = &cobra.Command{
	Use:     "describe",
	Short:   "Show details of Superplane resources",
	Long:    `Retrieve detailed information about Superplane resources.`,
	Aliases: []string{"desc", "get"},
}

func init() {
	RootCmd.AddCommand(describeCmd)

	// Canvas command
	describeCmd.AddCommand(describeCanvasCmd)
	describeCanvasCmd.Flags().String("name", "", "Name of the canvas (alternative to ID)")

	// Event Source command
	describeCmd.AddCommand(describeEventSourceCmd)
	describeEventSourceCmd.Flags().String("name", "", "Name of the event source (alternative to ID)")

	// Stage command
	describeCmd.AddCommand(describeStageCmd)
	describeStageCmd.Flags().String("name", "", "Name of the stage (alternative to ID)")
}
