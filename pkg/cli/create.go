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

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a resource from a file.",
	Long:  `Create a Superplane resource from a YAML file.`,

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
		case "Canvas":
			// Parse YAML to map
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			// Extract the name and requesterID from the YAML
			metadata, ok := yamlData["metadata"].(map[string]any)
			if !ok {
				Fail("Invalid Canvas YAML: metadata section missing")
			}

			name, ok := metadata["name"].(string)
			if !ok {
				Fail("Invalid Canvas YAML: name field missing")
			}

			// Create the canvas request
			request := openapi_client.NewSuperplaneCreateCanvasRequest()
			request.SetName(name)
			request.SetRequesterId(uuid.NewString())

			canvas, _, err := c.CanvasAPI.SuperplaneCreateCanvas(context.Background()).Body(*request).Execute()
			Check(err)

			fmt.Printf("Canvas '%s' created with ID '%s'.\n", *canvas.Canvas.Name, *canvas.Canvas.Id)

		case "EventSource":
			// Parse YAML to map
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			// Extract the metadata from the YAML
			metadata, ok := yamlData["metadata"].(map[string]any)
			if !ok {
				Fail("Invalid EventSource YAML: metadata section missing")
			}

			name, ok := metadata["name"].(string)
			if !ok {
				Fail("Invalid EventSource YAML: name field missing")
			}

			canvasID, ok := metadata["canvasId"].(string)
			if !ok {
				Fail("Invalid EventSource YAML: canvasId field missing")
			}

			// Create the event source request
			request := openapi_client.NewSuperplaneCreateEventSourceBody()
			request.SetName(name)
			request.SetRequesterId(uuid.NewString())
			response, _, err := c.EventSourceAPI.SuperplaneCreateEventSource(context.Background(), canvasID).Body(*request).Execute()
			Check(err)

			fmt.Printf("Event Source '%s' created with ID '%s'.\n",
				*response.EventSource.Name, *response.EventSource.Id)
			fmt.Printf("API Key: %s\n", *response.Key)
			fmt.Println("Save this key as it won't be shown again.")

		case "Stage":
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			metadata, ok := yamlData["metadata"].(map[string]any)
			if !ok {
				Fail("Invalid Stage YAML: metadata section missing")
			}

			name, ok := metadata["name"].(string)
			if !ok {
				Fail("Invalid Stage YAML: name missing")
			}

			canvasID, ok := metadata["canvasId"].(string)
			if !ok {
				Fail("Invalid Stage YAML: canvasId field missing")
			}

			spec, ok := yamlData["spec"].(map[string]any)
			if !ok {
				Fail("Invalid Stage YAML: spec section missing")
			}

			// Convert to JSON
			specData, err := json.Marshal(spec)
			Check(err)

			// Convert JSON to stage request
			var request openapi_client.SuperplaneCreateStageBody
			err = json.Unmarshal(specData, &request)
			Check(err)

			request.SetName(name)
			request.SetRequesterId(uuid.NewString())
			response, httpResponse, err := c.StageAPI.SuperplaneCreateStage(context.Background(), canvasID).
				Body(request).
				Execute()

			if err != nil {
				b, _ := io.ReadAll(httpResponse.Body)
				fmt.Printf("%s\n", string(b))
				os.Exit(1)
			}

			out, err := yaml.Marshal(response.Stage)
			Check(err)
			fmt.Printf("%s", string(out))

		default:
			Fail(fmt.Sprintf("Unsupported resource kind '%s'", kind))
		}
	},
}

func init() {
	RootCmd.AddCommand(createCmd)

	// File flag for root create command
	desc := "Filename, directory, or URL to files to use to create the resource"
	createCmd.Flags().StringP("file", "f", "", desc)
}
