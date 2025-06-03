package cli

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

var getCanvasCmd = &cobra.Command{
	Use:     "canvas [ID]",
	Short:   "Get canvas details",
	Long:    `Get details about a specific canvas`,
	Aliases: []string{"canvases"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		name, _ := cmd.Flags().GetString("name")

		c := DefaultClient()
		response, _, err := c.CanvasAPI.SuperplaneDescribeCanvas(context.Background(), id).Name(name).Execute()
		Check(err)

		out, err := yaml.Marshal(response.Canvas)
		Check(err)
		fmt.Printf("%s", string(out))
	},
}

var getEventSourceCmd = &cobra.Command{
	Use:     "event-source [ID]",
	Short:   "Get event source details",
	Long:    `Get details about a specific event source`,
	Aliases: []string{"event-sources", "eventsource", "eventsources"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		name, _ := cmd.Flags().GetString("name")
		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")

		c := DefaultClient()
		response, _, err := c.EventSourceAPI.SuperplaneDescribeEventSource(
			context.Background(),
			canvasIDOrName,
			id,
		).Name(name).Execute()
		Check(err)

		out, err := yaml.Marshal(response.EventSource)
		Check(err)
		fmt.Printf("%s", string(out))
	},
}

var getStageCmd = &cobra.Command{
	Use:     "stage [ID_OR_NAME]",
	Short:   "Get stage details",
	Long:    `Get details about a specific stage`,
	Aliases: []string{"stages"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		idOrName := args[0]

		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")

		c := DefaultClient()
		response, _, err := c.StageAPI.SuperplaneDescribeStage(
			context.Background(),
			canvasIDOrName,
			idOrName,
		).Name(idOrName).Execute()
		Check(err)

		out, err := yaml.Marshal(response.Stage)
		Check(err)
		fmt.Printf("%s", string(out))
	},
}

var getSecretCmd = &cobra.Command{
	Use:     "secret [ID_OR_NAME]",
	Short:   "Get secret details",
	Long:    `Get details about a specific secret`,
	Aliases: []string{"secrets"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		idOrName := args[0]
		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")

		c := DefaultClient()
		response, _, err := c.SecretAPI.SuperplaneDescribeSecret(
			context.Background(),
			canvasIDOrName,
			idOrName,
		).Execute()

		Check(err)

		out, err := yaml.Marshal(response.Secret)
		Check(err)
		fmt.Printf("%s", string(out))
	},
}

// Root describe command
var getCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show details of Superplane resources",
	Long:    `Get detailed information about Superplane resources.`,
	Aliases: []string{"desc", "get"},
}

func init() {
	RootCmd.AddCommand(getCmd)

	// Canvas command
	getCmd.AddCommand(getCanvasCmd)
	getCanvasCmd.Flags().String("name", "", "Name of the canvas (alternative to ID)")

	// Event Source command
	getCmd.AddCommand(getEventSourceCmd)
	getEventSourceCmd.Flags().String("name", "", "Name of the event source (alternative to ID)")
	getEventSourceCmd.Flags().String("canvas-id", "", "ID of the canvas (alternative to --canvas-name)")
	getEventSourceCmd.Flags().String("canvas-name", "", "Name of the canvas (alternative to --canvas-id)")

	// Stage command
	getCmd.AddCommand(getStageCmd)
	getStageCmd.Flags().String("canvas-id", "", "ID of the canvas (alternative to --canvas-name)")
	getStageCmd.Flags().String("canvas-name", "", "Name of the canvas (alternative to --canvas-id)")

	// Secret command
	getCmd.AddCommand(getSecretCmd)
	getSecretCmd.Flags().String("canvas-id", "", "ID of the canvas (alternative to --canvas-name)")
	getSecretCmd.Flags().String("canvas-name", "", "Name of the canvas (alternative to --canvas-id)")
}
