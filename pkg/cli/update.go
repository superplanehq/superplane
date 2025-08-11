package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/goccy/go-yaml"

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
		case "Secret":
			// Parse YAML to map
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			// Extract the metadata from the YAML
			metadata, ok := yamlData["metadata"].(map[string]any)
			if !ok {
				Fail("Invalid Secret YAML: metadata section missing")
			}

			domainId, ok := metadata["domainId"].(string)
			if !ok {
				Fail("Invalid Secret YAML: domainId field missing")
			}

			domainTypeFromYaml, ok := metadata["domainType"].(string)
			if !ok {
				Fail("Invalid Secret YAML: domainType field missing")
			}

			ID, ok := metadata["id"].(string)
			if !ok {
				Fail("Invalid Secret YAML: id field missing")
			}

			var secret openapi_client.SecretsSecret
			err = yaml.Unmarshal(data, &secret)
			Check(err)

			domainType, err := openapi_client.NewAuthorizationDomainTypeFromValue(domainTypeFromYaml)
			Check(err)

			response, httpResponse, err := c.SecretAPI.
				SecretsUpdateSecret(context.Background(), ID).
				Body(openapi_client.SecretsUpdateSecretBody{
					Secret:     &secret,
					DomainId:   &domainId,
					DomainType: domainType,
				}).
				Execute()

			if err != nil {
				b, _ := io.ReadAll(httpResponse.Body)
				fmt.Printf("%s\n", string(b))
				os.Exit(1)
			}

			out, err := yaml.Marshal(response.Secret)
			Check(err)
			fmt.Printf("%s", string(out))

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

			var stage openapi_client.SuperplaneStage
			err = yaml.Unmarshal(data, &stage)
			Check(err)

			// Execute request
			response, httpResponse, err := c.StageAPI.SuperplaneUpdateStage(context.Background(), canvasIDOrName, stageID).
				Body(openapi_client.SuperplaneUpdateStageBody{Stage: &stage}).
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

		case "ConnectionGroup":
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			metadata, ok := yamlData["metadata"].(map[string]any)
			if !ok {
				Fail("Invalid ConnectionGroup YAML: metadata section missing")
			}

			canvasIDOrName, ok := metadata["canvasId"].(string)
			if !ok {
				canvasIDOrName, ok = metadata["canvasName"].(string)
				if !ok {
					Fail("Invalid ConnectionGroup YAML: canvasId or canvasName field missing")
				}
			}

			ID, ok := metadata["id"].(string)
			if !ok {
				Fail("Invalid ConnectionGroup YAML: id field missing")
			}

			var connectionGroup openapi_client.SuperplaneConnectionGroup
			err = yaml.Unmarshal(data, &connectionGroup)
			Check(err)

			response, httpResponse, err := c.ConnectionGroupAPI.SuperplaneUpdateConnectionGroup(context.Background(), canvasIDOrName, ID).
				Body(openapi_client.SuperplaneUpdateConnectionGroupBody{ConnectionGroup: &connectionGroup}).
				Execute()

			if err != nil {
				body, err := io.ReadAll(httpResponse.Body)
				Check(err)
				fmt.Printf("Error: %v", err)
				fmt.Printf("HTTP Response: %s", string(body))
				os.Exit(1)
			}

			out, err := yaml.Marshal(response.ConnectionGroup)
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
		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name", true)
		stageIDOrName := getOneOrAnotherFlag(cmd, "stage-id", "stage-name", true)
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
	updateStageCmd.Flags().StringP("file", "f", "", "File containing stage configuration updates")
}
