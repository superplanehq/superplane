package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/superplanehq/superplane/pkg/cli/utils"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a resource from a file.",
	Long:  `Create a Superplane resource from a YAML file.`,

	Run: func(cmd *cobra.Command, args []string) {
		path, err := cmd.Flags().GetString("file")
		utils.CheckWithMessage(err, "Path not provided")

		// #nosec
		data, err := os.ReadFile(path)
		utils.CheckWithMessage(err, "Failed to read from resource file.")

		_, kind, err := utils.ParseYamlResourceHeaders(data)
		utils.Check(err)

		c := DefaultClient()

		switch strings.ToLower(kind) {
		case "canvas":
			// Parse YAML to map
			var yamlData map[string]interface{}
			err = yaml.Unmarshal(data, &yamlData)
			utils.Check(err)

			// Extract the name and requesterID from the YAML
			metadata, ok := yamlData["metadata"].(map[interface{}]interface{})
			if !ok {
				utils.Fail("Invalid Canvas YAML: metadata section missing")
			}

			name, ok := metadata["name"].(string)
			if !ok {
				utils.Fail("Invalid Canvas YAML: name field missing")
			}

			requesterID, _ := metadata["requesterId"].(string)

			// Create the canvas request
			request := openapi_client.NewSuperplaneCreateCanvasRequest()
			request.SetName(name)
			if requesterID != "" {
				request.SetRequesterId(requesterID)
			}

			canvas, _, err := c.CanvasAPI.SuperplaneCreateCanvas(context.Background()).Body(*request).Execute()
			utils.Check(err)

			fmt.Printf("Canvas '%s' created with ID '%s'.\n", *canvas.Canvas.Name, *canvas.Canvas.Id)

		case "eventsource":
			// Parse YAML to map
			var yamlData map[string]interface{}
			err = yaml.Unmarshal(data, &yamlData)
			utils.Check(err)

			// Extract the metadata from the YAML
			metadata, ok := yamlData["metadata"].(map[interface{}]interface{})
			if !ok {
				utils.Fail("Invalid EventSource YAML: metadata section missing")
			}

			name, ok := metadata["name"].(string)
			if !ok {
				utils.Fail("Invalid EventSource YAML: name field missing")
			}

			canvasID, ok := metadata["canvasId"].(string)
			if !ok {
				utils.Fail("Invalid EventSource YAML: canvasId field missing")
			}

			requesterID, _ := metadata["requesterId"].(string)

			// Create the event source request
			request := openapi_client.NewSuperplaneCreateEventSourceBody()
			request.SetName(name)
			if requesterID != "" {
				request.SetRequesterId(requesterID)
			}

			response, _, err := c.EventSourceAPI.SuperplaneCreateEventSource(context.Background(), canvasID).Body(*request).Execute()
			utils.Check(err)

			fmt.Printf("Event Source '%s' created with ID '%s'.\n",
				*response.EventSource.Name, *response.EventSource.Id)
			fmt.Printf("API Key: %s\n", *response.Key)
			fmt.Println("Save this key as it won't be shown again.")

		case "stage":
			// Parse YAML to map first
			var yamlData map[string]interface{}
			err = yaml.Unmarshal(data, &yamlData)
			utils.Check(err)

			// Extract metadata
			metadata, ok := yamlData["metadata"].(map[interface{}]interface{})
			if !ok {
				utils.Fail("Invalid Stage YAML: metadata section missing")
			}

			canvasID, ok := metadata["canvasId"].(string)
			if !ok {
				utils.Fail("Invalid Stage YAML: canvasId field missing")
			}

			// Convert to JSON
			jsonData, err := json.Marshal(yamlData)
			utils.Check(err)

			// Convert JSON to stage request
			var stageBody openapi_client.SuperplaneCreateStageBody
			err = json.Unmarshal(jsonData, &stageBody)
			utils.Check(err)

			response, _, err := c.StageAPI.SuperplaneCreateStage(context.Background(), canvasID).Body(stageBody).Execute()
			utils.Check(err)

			fmt.Printf("Stage '%s' created with ID '%s' in Canvas '%s'.\n",
				*response.Stage.Name, *response.Stage.Id, *response.Stage.CanvasId)

		default:
			utils.Fail(fmt.Sprintf("Unsupported resource kind '%s'", kind))
		}
	},
}

var createCanvasCmd = &cobra.Command{
	Use:     "canvas [NAME]",
	Short:   "Create a new canvas",
	Long:    `Create a new canvas with the specified name`,
	Aliases: []string{"canvases"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		requesterID, _ := cmd.Flags().GetString("requester-id")

		c := DefaultClient()
		request := openapi_client.NewSuperplaneCreateCanvasRequest()
		request.SetName(name)
		request.SetRequesterId(requesterID)

		canvas, _, err := c.CanvasAPI.SuperplaneCreateCanvas(context.Background()).Body(*request).Execute()
		utils.Check(err)

		fmt.Printf("Canvas '%s' created with ID '%s'.\n", *canvas.Canvas.Name, *canvas.Canvas.Id)
	},
}

var createEventSourceCmd = &cobra.Command{
	Use:     "event-source [CANVAS_ID] [NAME]",
	Short:   "Create a new event source",
	Long:    `Create a new event source for the specified canvas`,
	Aliases: []string{"event-sources", "eventsource", "eventsources"},
	Args:    cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {
		canvasID := args[0]
		name := args[1]
		requesterID, _ := cmd.Flags().GetString("requester-id")

		c := DefaultClient()
		request := openapi_client.NewSuperplaneCreateEventSourceBody()
		request.SetName(name)
		request.SetRequesterId(requesterID)

		response, _, err := c.EventSourceAPI.SuperplaneCreateEventSource(context.Background(), canvasID).Body(*request).Execute()
		utils.Check(err)

		fmt.Printf("Event Source '%s' created with ID '%s'.\n", *response.EventSource.Name, *response.EventSource.Id)
		fmt.Printf("API Key: %s\n", *response.Key)
		fmt.Println("Save this key as it won't be shown again.")
	},
}

var createStageCmd = &cobra.Command{
	Use:     "stage [CANVAS_ID] [NAME]",
	Short:   "Create a new stage",
	Long:    `Create a new stage for the specified canvas`,
	Aliases: []string{"stages"},
	Args:    cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {
		canvasID := args[0]
		name := args[1]
		requesterID, _ := cmd.Flags().GetString("requester-id")
		yamlFile, _ := cmd.Flags().GetString("file")

		c := DefaultClient()

		if yamlFile != "" {
			// #nosec
			data, err := os.ReadFile(yamlFile)
			utils.CheckWithMessage(err, "Failed to read from stage configuration file.")

			// Parse YAML to map first
			var yamlData map[string]interface{}
			err = yaml.Unmarshal(data, &yamlData)
			utils.Check(err)

			// Convert to JSON
			jsonData, err := json.Marshal(yamlData)
			utils.Check(err)

			// Convert JSON to stage request
			var stageBody openapi_client.SuperplaneCreateStageBody
			err = json.Unmarshal(jsonData, &stageBody)
			utils.Check(err)

			// Override name if provided in command
			if name != "" {
				stageBody.SetName(name)
			}

			// Set requesterID
			stageBody.SetRequesterId(requesterID)

			response, _, err := c.StageAPI.SuperplaneCreateStage(context.Background(), canvasID).Body(stageBody).Execute()
			utils.Check(err)

			fmt.Printf("Stage '%s' created with ID '%s' in Canvas '%s'.\n",
				*response.Stage.Name, *response.Stage.Id, *response.Stage.CanvasId)
			return
		}

		// If no YAML file provided, create a basic stage
		request := openapi_client.NewSuperplaneCreateStageBody()
		request.SetName(name)
		request.SetRequesterId(requesterID)

		response, _, err := c.StageAPI.SuperplaneCreateStage(context.Background(), canvasID).Body(*request).Execute()
		utils.Check(err)

		fmt.Printf("Stage '%s' created with ID '%s'.\n", *response.Stage.Name, *response.Stage.Id)
	},
}

func init() {
	RootCmd.AddCommand(createCmd)

	// Canvas command
	createCmd.AddCommand(createCanvasCmd)
	createCanvasCmd.Flags().String("requester-id", "", "ID of the user creating the canvas")

	// Event Source command
	createCmd.AddCommand(createEventSourceCmd)
	createEventSourceCmd.Flags().String("requester-id", "", "ID of the user creating the event source")

	// Stage command
	createCmd.AddCommand(createStageCmd)
	createStageCmd.Flags().String("requester-id", "", "ID of the user creating the stage")
	createStageCmd.Flags().StringP("file", "f", "", "File containing stage configuration")

	// File flag for root create command
	desc := "Filename, directory, or URL to files to use to create the resource"
	createCmd.Flags().StringP("file", "f", "", desc)
}
