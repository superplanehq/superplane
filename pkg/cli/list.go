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
		c := DefaultClient()
		response, _, err := c.CanvasAPI.SuperplaneListCanvases(context.Background()).Execute()
		Check(err)

		if len(response.Canvases) == 0 {
			fmt.Println("No canvases found.")
			return
		}

		for i, canvas := range response.Canvases {
			fmt.Printf("%d. %s (ID: %s)\n", i+1, *canvas.GetMetadata().Name, *canvas.GetMetadata().Id)
			fmt.Printf("   Created at: %s\n", *canvas.GetMetadata().CreatedAt)
			fmt.Printf("   Created by: %s\n", *canvas.GetMetadata().CreatedBy)

			if i < len(response.Canvases)-1 {
				fmt.Println()
			}
		}
	},
}

var listEventSourcesCmd = &cobra.Command{
	Use:     "event-sources",
	Short:   "List all event sources for a canvas",
	Long:    `Retrieve a list of all event sources for the specified canvas`,
	Aliases: []string{"eventsources"},
	Args:    cobra.ExactArgs(0),

	Run: func(cmd *cobra.Command, args []string) {
		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")

		c := DefaultClient()
		response, _, err := c.EventSourceAPI.SuperplaneListEventSources(context.Background(), canvasIDOrName).Execute()
		Check(err)

		if len(response.EventSources) == 0 {
			fmt.Println("No event sources found for this canvas.")
			return
		}

		fmt.Printf("Found %d event sources:\n\n", len(response.EventSources))
		for i, es := range response.EventSources {
			fmt.Printf("%d. %s (ID: %s)\n", i+1, *es.GetMetadata().Name, *es.GetMetadata().Id)
			fmt.Printf("   Canvas: %s\n", *es.GetMetadata().CanvasId)
			fmt.Printf("   Created at: %s\n", *es.GetMetadata().CreatedAt)

			if i < len(response.EventSources)-1 {
				fmt.Println()
			}
		}
	},
}

var listStagesCmd = &cobra.Command{
	Use:     "stages",
	Short:   "List all stages for a canvas",
	Long:    `Retrieve a list of all stages for the specified canvas`,
	Aliases: []string{"stages"},
	Args:    cobra.ExactArgs(0),

	Run: func(cmd *cobra.Command, args []string) {
		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")

		c := DefaultClient()
		response, _, err := c.StageAPI.SuperplaneListStages(context.Background(), canvasIDOrName).Execute()
		Check(err)

		if len(response.Stages) == 0 {
			fmt.Println("No stages found for this canvas.")
			return
		}

		fmt.Printf("Found %d stages:\n\n", len(response.Stages))
		for i, stage := range response.Stages {
			fmt.Printf("%d. %s (ID: %s)\n", i+1, *stage.GetMetadata().Name, *stage.GetMetadata().Id)
			fmt.Printf("   Canvas: %s\n", *stage.GetMetadata().CanvasId)
			fmt.Printf("   Created at: %s\n", *stage.GetMetadata().CreatedAt)

			if i < len(response.Stages)-1 {
				fmt.Println()
			}
		}
	},
}

var listSecretsCmd = &cobra.Command{
	Use:     "secrets",
	Short:   "List all secrets for a canvas",
	Long:    `Retrieve a list of all secrets for the specified canvas`,
	Aliases: []string{"secrets"},
	Args:    cobra.ExactArgs(0),

	Run: func(cmd *cobra.Command, args []string) {
		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")

		c := DefaultClient()
		response, _, err := c.SecretAPI.SuperplaneListSecrets(context.Background(), canvasIDOrName).Execute()
		Check(err)

		if len(response.Secrets) == 0 {
			fmt.Println("No secrets found for this canvas.")
			return
		}

		fmt.Printf("Found %d secrets:\n\n", len(response.Secrets))
		for i, secret := range response.Secrets {
			fmt.Printf("%d. %s (ID: %s)\n", i+1, *secret.GetMetadata().Name, *secret.GetMetadata().Id)
			fmt.Printf("   Canvas: %s\n", *secret.GetMetadata().CanvasId)
			fmt.Printf("   Provider: %s\n", string(*secret.GetSpec().Provider))
			fmt.Printf("   Created at: %s\n", *secret.GetMetadata().CreatedAt)

			if secret.GetSpec().Local != nil && secret.GetSpec().Local.Data != nil {
				fmt.Println("   Values:")
				for k, v := range *secret.GetSpec().Local.Data {
					fmt.Printf("     %s = %s\n", k, v)
				}
			}

			if i < len(response.Secrets)-1 {
				fmt.Println()
			}
		}
	},
}

var listEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List stage events",
	Long:  `List all events for a specific stage`,
	Args:  cobra.ExactArgs(0),

	Run: func(cmd *cobra.Command, args []string) {
		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")
		stageIDOrName := getOneOrAnotherFlag(cmd, "stage-id", "stage-name")

		states, _ := cmd.Flags().GetStringSlice("states")
		stateReasons, _ := cmd.Flags().GetStringSlice("state-reasons")

		c := DefaultClient()
		listRequest := c.EventAPI.SuperplaneListStageEvents(context.Background(), canvasIDOrName, stageIDOrName)

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

			if len(event.Inputs) > 0 {
				fmt.Println("   Inputs:")
				for _, input := range event.Inputs {
					fmt.Printf("     * %s = %s\n", *input.Name, *input.Value)
				}
			}

			if event.Execution != nil {
				fmt.Println("   Execution:")
				fmt.Printf("      ID: %s\n", *event.Execution.Id)
				fmt.Printf("      Reference ID: %s\n", *event.Execution.ReferenceId)
				fmt.Printf("      State: %s\n", *event.Execution.State)
				fmt.Printf("      Result: %s\n", *event.Execution.Result)
				fmt.Printf("      Created at: %s\n", event.Execution.CreatedAt)
				fmt.Printf("      Started at: %s\n", event.Execution.StartedAt)
				fmt.Printf("      Finished at: %s\n", event.Execution.FinishedAt)
				if len(event.Execution.Outputs) > 0 {
					fmt.Println("      Outputs:")
					for _, output := range event.Execution.Outputs {
						fmt.Printf("        * %s = %s\n", *output.Name, *output.Value)
					}
				}
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
	listEventSourcesCmd.Flags().String("canvas-id", "", "Canvas ID")
	listEventSourcesCmd.Flags().String("canvas-name", "", "Canvas name")

	// Stages command
	listCmd.AddCommand(listStagesCmd)
	listStagesCmd.Flags().String("canvas-id", "", "Canvas ID")
	listStagesCmd.Flags().String("canvas-name", "", "Canvas name")

	// Secrets command
	listCmd.AddCommand(listSecretsCmd)
	listSecretsCmd.Flags().String("canvas-id", "", "Canvas ID")
	listSecretsCmd.Flags().String("canvas-name", "", "Canvas name")

	// Events command
	listCmd.AddCommand(listEventsCmd)
	listEventsCmd.Flags().StringSlice("states", []string{}, "Filter by event states (PENDING, WAITING, PROCESSED)")
	listEventsCmd.Flags().StringSlice("state-reasons", []string{}, "Filter by event state reasons")
	listEventsCmd.Flags().String("canvas-id", "", "Canvas ID")
	listEventsCmd.Flags().String("canvas-name", "", "Canvas name")
	listEventsCmd.Flags().String("stage-id", "", "Stage ID")
	listEventsCmd.Flags().String("stage-name", "", "Stage name")
}
