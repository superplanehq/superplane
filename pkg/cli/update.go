package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/google/uuid"

	"github.com/spf13/cobra"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a resource from a file.",
	Long:    `Update a Superplane resource from a YAML file.`,
	Aliases: []string{"update", "edit"},

	Run: func(cmd *cobra.Command, args []string) {
		path, err := cmd.Flags().GetString("file")
		CheckWithMessage(err, "Path not provided")

		// #nosec
		data, err := os.ReadFile(path)
		CheckWithMessage(err, "Failed to read from resource file.")

		_, kind, err := ParseYamlResourceHeaders(data)
		Check(err)

		c := DefaultClient()

		switch kind {
		case "Stage":
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			metadata, ok := yamlData["metadata"].(map[string]any)
			if !ok {
				Fail("Invalid Stage YAML: metadata section missing")
			}

			canvasID, ok := metadata["canvasId"].(string)
			if !ok {
				Fail("Invalid Stage YAML: canvasId field missing")
			}

			stageID, ok := metadata["id"].(string)
			if !ok {
				Fail("Invalid Stage YAML: id field missing")
			}

			spec, ok := yamlData["spec"].(map[string]any)
			if !ok {
				Fail("Invalid Stage YAML: spec section missing")
			}

			// Convert to JSON
			specData, err := json.Marshal(spec)
			Check(err)

			// Convert JSON to stage request
			var request openapi_client.SuperplaneUpdateStageBody
			err = json.Unmarshal(specData, &request)
			Check(err)

			// TODO: this should be known through the API token used to call the API
			// so we just put something here until we have auth in this API.
			request.SetRequesterId(uuid.NewString())

			_, _, err = c.StageAPI.SuperplaneUpdateStage(context.Background(), canvasID, stageID).
				Body(request).
				Execute()

			Check(err)

			fmt.Printf("Stage '%s' updated successfully.\n", stageID)

		default:
			Fail(fmt.Sprintf("Unsupported resource kind '%s' for update", kind))
		}
	},
}

var updateStageCmd = &cobra.Command{
	Use:   "stage [CANVAS_ID] [STAGE_ID]",
	Short: "Update a stage's configuration",
	Long:  `Update a stage's configuration, such as its connections.`,
	Args:  cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {
		canvasID := args[0]
		stageID := args[1]
		requesterID, _ := cmd.Flags().GetString("requester-id")
		yamlFile, _ := cmd.Flags().GetString("file")

		if yamlFile == "" {
			fmt.Println("Error: You must specify a configuration file with --file")
			os.Exit(1)
		}

		data, err := os.ReadFile(yamlFile)
		CheckWithMessage(err, "Failed to read from stage configuration file.")

		var yamlData map[string]interface{}
		err = yaml.Unmarshal(data, &yamlData)
		Check(err)

		var connections []interface{}
		if spec, ok := yamlData["spec"].(map[interface{}]interface{}); ok {
			if conns, ok := spec["connections"].([]interface{}); ok {
				connections = conns
			}
		}

		// Create update request
		request := openapi_client.NewSuperplaneUpdateStageBody()
		request.SetRequesterId(requesterID)

		if len(connections) > 0 {
			connJSON, err := json.Marshal(connections)
			Check(err)

			var apiConnections []openapi_client.SuperplaneConnection
			err = json.Unmarshal(connJSON, &apiConnections)
			Check(err)

			request.SetConnections(apiConnections)
		}

		c := DefaultClient()
		_, _, err = c.StageAPI.SuperplaneUpdateStage(
			context.Background(),
			canvasID,
			stageID,
		).Body(*request).Execute()
		Check(err)

		fmt.Printf("Stage '%s' updated successfully.\n", stageID)
	},
}

func init() {
	RootCmd.AddCommand(updateCmd)

	// File flag for root update command
	desc := "Filename, directory, or URL to files to use to update the resource"
	updateCmd.Flags().StringP("file", "f", "", desc)

	// Stage command
	updateCmd.AddCommand(updateStageCmd)
	updateStageCmd.Flags().String("requester-id", "", "ID of the user updating the stage")
	updateStageCmd.Flags().StringP("file", "f", "", "File containing stage configuration updates")
}
