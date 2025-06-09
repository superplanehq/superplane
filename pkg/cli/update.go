package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

			canvasIDOrName, ok := metadata["canvasId"].(string)
			if !ok {
				canvasIDOrName, ok = metadata["canvasName"].(string)
				if !ok {
					Fail("Invalid Stage YAML: canvasId or canvasName field missing")
				}
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

			response, httpResponse, err := c.StageAPI.SuperplaneUpdateStage(context.Background(), canvasIDOrName, stageID).
				Body(request).
				Execute()

			if err != nil {
				body, err := io.ReadAll(httpResponse.Body)
				Check(err)
				fmt.Printf("Error: %v", err)
				fmt.Printf("HTTP Response: %s", string(body))
				os.Exit(1)
			}

			out, err := yaml.Marshal(response.Stage)
			Check(err)
			fmt.Printf("%s", string(out))

		default:
			Fail(fmt.Sprintf("Unsupported resource kind '%s' for update", kind))
		}
	},
}

var updateStageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Update a stage's configuration",
	Long:  `Update a stage's configuration, such as its connections.`,
	Args:  cobra.ExactArgs(0),

	Run: func(cmd *cobra.Command, args []string) {
		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")
		stageIDOrName := getOneOrAnotherFlag(cmd, "stage-id", "stage-name")
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

		// Create update request with nested structure
		request := openapi_client.NewSuperplaneUpdateStageBody()
		request.SetRequesterId(requesterID)

		// Create stage with spec
		stage := openapi_client.NewSuperplaneStage()
		
		// Create stage spec
		stageSpec := openapi_client.NewSuperplaneStageSpec()
		
		// Parse connections if present
		if len(connections) > 0 {
			connJSON, err := json.Marshal(connections)
			Check(err)

			var apiConnections []openapi_client.SuperplaneConnection
			err = json.Unmarshal(connJSON, &apiConnections)
			Check(err)

			// Set connections in spec
			stageSpec.SetConnections(apiConnections)
		}
		
		// Set spec in stage
		stage.SetSpec(*stageSpec)
		
		// Set stage in request
		request.SetStage(*stage)

		c := DefaultClient()
		_, _, err = c.StageAPI.SuperplaneUpdateStage(
			context.Background(),
			canvasIDOrName,
			stageIDOrName,
		).Body(*request).Execute()
		Check(err)

		fmt.Printf("Stage '%s' updated successfully.\n", stageIDOrName)
	},
}

func init() {
	RootCmd.AddCommand(updateCmd)

	// File flag for root update command
	desc := "Filename, directory, or URL to files to use to update the resource"
	updateCmd.Flags().StringP("file", "f", "", desc)

	// Stage command
	updateCmd.AddCommand(updateStageCmd)
	updateStageCmd.Flags().String("canvas-id", "", "Canvas ID")
	updateStageCmd.Flags().String("canvas-name", "", "Canvas name")
	updateStageCmd.Flags().String("stage-id", "", "Stage ID")
	updateStageCmd.Flags().String("stage-name", "", "Stage name")
	updateStageCmd.Flags().String("requester-id", "", "ID of the user updating the stage")
	updateStageCmd.Flags().StringP("file", "f", "", "File containing stage configuration updates")
}
