package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var listCanvasesCmd = &cobra.Command{
	Use:   "canvases",
	Short: "List all canvases",
	Long:  `Retrieve a list of all canvases`,
	Args:  cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		// Note: The OpenAPI spec doesn't have a list canvases endpoint
		// This is a placeholder for when that endpoint is added
		fmt.Println("Listing canvases operation is not available in the current API version.")
	},
}

var listEventSourcesCmd = &cobra.Command{
	Use:     "event-sources [CANVAS_ID]",
	Short:   "List all event sources for a canvas",
	Long:    `Retrieve a list of all event sources for the specified canvas`,
	Aliases: []string{"eventsources"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		canvasID := args[0]

		c := DefaultClient()
		response, _, err := c.EventSourceAPI.SuperplaneListEventSources(context.Background(), canvasID).Execute()
		Check(err)

		if len(response.EventSources) == 0 {
			fmt.Println("No event sources found for this canvas.")
			return
		}

		fmt.Printf("Found %d event sources:\n\n", len(response.EventSources))
		for i, es := range response.EventSources {
			fmt.Printf("%d. %s (ID: %s)\n", i+1, *es.Name, *es.Id)
			fmt.Printf("   Canvas: %s\n", *es.CanvasId)
			fmt.Printf("   Created at: %s\n", *es.CreatedAt)

			if i < len(response.EventSources)-1 {
				fmt.Println()
			}
		}
	},
}

var listStagesCmd = &cobra.Command{
	Use:   "stages [CANVAS_ID]",
	Short: "List all stages for a canvas",
	Long:  `Retrieve a list of all stages for the specified canvas`,
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		canvasID := args[0]

		c := DefaultClient()
		response, _, err := c.StageAPI.SuperplaneListStages(context.Background(), canvasID).Execute()
		Check(err)

		if len(response.Stages) == 0 {
			fmt.Println("No stages found for this canvas.")
			return
		}

		fmt.Printf("Found %d stages:\n\n", len(response.Stages))
		for i, stage := range response.Stages {
			fmt.Printf("%d. %s (ID: %s)\n", i+1, *stage.Name, *stage.Id)
			fmt.Printf("   Canvas: %s\n", *stage.CanvasId)
			fmt.Printf("   Created at: %s\n", *stage.CreatedAt)

			if i < len(response.Stages)-1 {
				fmt.Println()
			}
		}
	},
}

var listEventsCmd = &cobra.Command{
	Use:   "events [CANVAS_ID] [STAGE_ID]",
	Short: "List stage events",
	Long:  `List all events for a specific stage`,
	Args:  cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {
		canvasID := args[0]
		stageID := args[1]

		states, _ := cmd.Flags().GetStringSlice("states")
		stateReasons, _ := cmd.Flags().GetStringSlice("state-reasons")

		c := DefaultClient()
		listRequest := c.EventAPI.SuperplaneListStageEvents(context.Background(), canvasID, stageID)

		if len(states) > 0 {
			listRequest = listRequest.States(states)
		}
		if len(stateReasons) > 0 {
			listRequest = listRequest.StateReasons(stateReasons)
		}

		response, _, err := listRequest.Execute()
		Check(err)

		if len(response.Events) == 0 {
			fmt.Println("No events found.")
			return
		}

		fmt.Printf("Found %d events:\n\n", len(response.Events))
		for i, event := range response.Events {
			fmt.Printf("%d. Event ID: %s\n", i+1, *event.Id)
			fmt.Printf("   Source: %s (%s)\n", *event.SourceId, *event.SourceType)
			fmt.Printf("   State: %s (%s)\n", *event.State, *event.StateReason)
			fmt.Printf("   Created: %s\n", *event.CreatedAt)

			if event.Execution != nil {
				fmt.Println("   Execution:")
				fmt.Printf("      ID: %s\n", *event.Execution.Id)
				fmt.Printf("      Reference ID: %s\n", *event.Execution.ReferenceId)
				fmt.Printf("      State: %s\n", *event.Execution.State)
				fmt.Printf("      Result: %s\n", *event.Execution.Result)
				fmt.Printf("      Created at: %s\n", event.Execution.CreatedAt)
				fmt.Printf("      Started at: %s\n", event.Execution.StartedAt)
				fmt.Printf("      Finished at: %s\n", event.Execution.FinishedAt)
			}

			if len(event.Approvals) > 0 {
				fmt.Println("   Approvals:")
				for j, approval := range event.Approvals {
					fmt.Printf("     %d. By: %s at %s\n", j+1, *approval.ApprovedBy, *approval.ApprovedAt)
				}
			}

			if i < len(response.Events)-1 {
				fmt.Println()
			}
		}
	},
}

// Root list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List Superplane resources",
	Long:    `List multiple Superplane resources.`,
	Aliases: []string{"ls"},
}

func init() {
	RootCmd.AddCommand(listCmd)

	// Canvases command
	listCmd.AddCommand(listCanvasesCmd)

	// Event Sources command
	listCmd.AddCommand(listEventSourcesCmd)

	// Stages command
	listCmd.AddCommand(listStagesCmd)

	// Events command
	listCmd.AddCommand(listEventsCmd)
	listEventsCmd.Flags().StringSlice("states", []string{}, "Filter by event states (PENDING, WAITING, PROCESSED)")
	listEventsCmd.Flags().StringSlice("state-reasons", []string{}, "Filter by event state reasons")
}
