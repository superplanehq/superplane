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

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a resource from a file.",
	Long:    `Update a Superplane resource from a YAML file.`,
	Aliases: []string{"update", "edit"},

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
		case "stage":
			var yamlData map[string]interface{}
			err = yaml.Unmarshal(data, &yamlData)
			utils.Check(err)

			metadata, ok := yamlData["metadata"].(map[interface{}]interface{})
			if !ok {
				utils.Fail("Invalid Stage YAML: metadata section missing")
			}

			canvasID, ok := metadata["canvasId"].(string)
			if !ok {
				utils.Fail("Invalid Stage YAML: canvasId field missing")
			}

			stageID, ok := metadata["id"].(string)
			if !ok {
				utils.Fail("Invalid Stage YAML: id field missing")
			}

			requesterID, _ := metadata["requesterId"].(string)

			var connections []interface{}
			if spec, ok := yamlData["spec"].(map[interface{}]interface{}); ok {
				if conns, ok := spec["connections"].([]interface{}); ok {
					connections = conns
				}
			}

			request := openapi_client.NewSuperplaneUpdateStageBody()
			if requesterID != "" {
				request.SetRequesterId(requesterID)
			}

			if len(connections) > 0 {
				connJSON, err := json.Marshal(connections)
				utils.Check(err)

				var apiConnections []openapi_client.SuperplaneConnection
				err = json.Unmarshal(connJSON, &apiConnections)
				utils.Check(err)

				request.SetConnections(apiConnections)
			}

			_, _, err = c.StageAPI.SuperplaneUpdateStage(
				context.Background(),
				canvasID,
				stageID,
			).Body(*request).Execute()
			utils.Check(err)

			fmt.Printf("Stage '%s' updated successfully.\n", stageID)

		default:
			utils.Fail(fmt.Sprintf("Unsupported resource kind '%s' for update", kind))
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
		utils.CheckWithMessage(err, "Failed to read from stage configuration file.")

		var yamlData map[string]interface{}
		err = yaml.Unmarshal(data, &yamlData)
		utils.Check(err)

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
			utils.Check(err)

			var apiConnections []openapi_client.SuperplaneConnection
			err = json.Unmarshal(connJSON, &apiConnections)
			utils.Check(err)

			request.SetConnections(apiConnections)
		}

		c := DefaultClient()
		_, _, err = c.StageAPI.SuperplaneUpdateStage(
			context.Background(),
			canvasID,
			stageID,
		).Body(*request).Execute()
		utils.Check(err)

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
