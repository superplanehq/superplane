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
			
			// Create Canvas with metadata
			canvas := openapi_client.NewSuperplaneCanvas()
			canvasMeta := openapi_client.NewSuperplaneCanvasMetadata()
			canvasMeta.SetName(name)
			canvas.SetMetadata(*canvasMeta)
			
			// Set canvas in request
			request.SetCanvas(*canvas)
			
			// Set requester ID
			requesterId := uuid.NewString()
			request.SetRequesterId(requesterId)

			response, _, err := c.CanvasAPI.SuperplaneCreateCanvas(context.Background()).Body(*request).Execute()
			Check(err)

			// Access the returned canvas
			canvasResult := response.GetCanvas()
			fmt.Printf("Canvas '%s' created with ID '%s'.\n", *canvasResult.GetMetadata().Name, *canvasResult.GetMetadata().Id)

		case "Secret":
			// Parse YAML to map
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			// Extract the metadata from the YAML
			metadata, ok := yamlData["metadata"].(map[string]interface{})
			if !ok {
				Fail("Invalid Secret YAML: metadata section missing")
			}

			name, ok := metadata["name"].(string)
			if !ok {
				Fail("Invalid Secret YAML: name field missing")
			}

			canvasID, ok := metadata["canvasId"].(string)
			if !ok {
				Fail("Invalid Secret YAML: canvasId field missing")
			}

			spec, ok := yamlData["spec"].(map[string]interface{})
			if !ok {
				Fail("Invalid Secret YAML: spec section missing")
			}

			// Prepare request

			// Create the initial request
			request := openapi_client.NewSuperplaneCreateSecretBody()
			
			// Create Secret with metadata and spec
			secret := openapi_client.NewSuperplaneSecret()
			secretMeta := openapi_client.NewSuperplaneSecretMetadata()
			secretMeta.SetName(name)
			secretMeta.SetCanvasId(canvasID)
			secret.SetMetadata(*secretMeta)
			
			// Create a proper secret spec from the YAML data
			secretSpec := openapi_client.NewSuperplaneSecretSpec()
			
			// Convert provider string to enum if present
			if providerStr, ok := spec["provider"].(string); ok {
				var provider openapi_client.SecretProvider
				switch providerStr {
				case "PROVIDER_LOCAL":
					provider = openapi_client.SECRETPROVIDER_PROVIDER_LOCAL
				}
				secretSpec.SetProvider(provider)
			}
			
			// Handle local data if present
			if localData, ok := spec["local"].(map[string]interface{}); ok {
				local := openapi_client.NewSecretLocal()
				if dataMap, ok := localData["data"].(map[string]interface{}); ok {
					// Convert to string map
					stringMap := make(map[string]string)
					for k, v := range dataMap {
						if strVal, ok := v.(string); ok {
							stringMap[k] = strVal
						}
					}
					local.SetData(stringMap)
				}
				secretSpec.SetLocal(*local)
			}
			
			// Set the spec
			secret.SetSpec(*secretSpec)
			
			// Add secret to request and set requester ID
			request.SetSecret(*secret)
			requesterId := uuid.NewString()
			request.SetRequesterId(requesterId)

			// Send request
			response, httpResponse, err := c.SecretAPI.SuperplaneCreateSecret(context.Background(), canvasID).
				Body(*request).
				Execute()

			if err != nil {
				b, _ := io.ReadAll(httpResponse.Body)
				fmt.Printf("%s\n", string(b))
				os.Exit(1)
			}

			out, err := yaml.Marshal(response.Secret)
			Check(err)
			fmt.Printf("%s", string(out))

		case "EventSource":
			// Parse YAML to map
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			// Extract the metadata from the YAML
			metadata, ok := yamlData["metadata"].(map[string]interface{})
			if !ok {
				Fail("Invalid EventSource YAML: metadata section missing")
			}

			name, ok := metadata["name"].(string)
			if !ok {
				Fail("Invalid EventSource YAML: name field missing")
			}

			canvasIDOrName, ok := metadata["canvasId"].(string)
			if !ok {
				canvasIDOrName, ok = metadata["canvasName"].(string)
				if !ok {
					Fail("Invalid EventSource YAML: canvasId or canvasName field missing")
				}
			}

			// Create the event source request
			request := openapi_client.NewSuperplaneCreateEventSourceBody()
			
			// Create EventSource with metadata and spec
			eventSource := openapi_client.NewSuperplaneEventSource()
			esMeta := openapi_client.NewSuperplaneEventSourceMetadata()
			esMeta.SetName(name)
			esMeta.SetCanvasId(canvasIDOrName)
			eventSource.SetMetadata(*esMeta)
			
			// Create an empty spec for the EventSource
			emptySpec := make(map[string]interface{})
			eventSource.SetSpec(emptySpec)
			
			// Set in request
			request.SetEventSource(*eventSource)
			requesterId := uuid.NewString()
			request.SetRequesterId(requesterId)
			response, _, err := c.EventSourceAPI.SuperplaneCreateEventSource(context.Background(), canvasIDOrName).Body(*request).Execute()
			Check(err)

			// Access the event source from response
			es := response.GetEventSource()
			fmt.Printf("Event Source '%s' created with ID '%s'.\n",
				*es.GetMetadata().Name, *es.GetMetadata().Id)
			fmt.Printf("Key: %s\n", *response.Key)
			fmt.Println("! Save this key as it won't be shown again.")

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

			canvasIDOrName, ok := metadata["canvasId"].(string)
			if !ok {
				canvasIDOrName, ok = metadata["canvasName"].(string)
				if !ok {
					Fail("Invalid Stage YAML: canvasId or canvasName field missing")
				}
			}

			spec, ok := yamlData["spec"].(map[string]any)
			if !ok {
				Fail("Invalid Stage YAML: spec section missing")
			}

			// Convert to JSON not needed anymore
			// We can use the spec map directly

			// Keep using the original workflow for stages
			// Parse the stage spec directly from YAML
			// instead of trying to extract it from a nested map
			
			// Create stage with metadata and spec
			stage := openapi_client.NewSuperplaneStage()
			stageMeta := openapi_client.NewSuperplaneStageMetadata()
			stageMeta.SetName(name)
			stageMeta.SetCanvasId(canvasIDOrName)
			stage.SetMetadata(*stageMeta)
			
			// Convert the spec to JSON
			specData, err := json.Marshal(spec)
			Check(err)
			
			// Parse into the proper struct
			var stageSpec openapi_client.SuperplaneStageSpec
			err = json.Unmarshal(specData, &stageSpec)
			Check(err)
			
			// Set the spec
			stage.SetSpec(stageSpec)
			
			// Create request and set stage
			request := openapi_client.NewSuperplaneCreateStageBody()
			request.SetStage(*stage)
			requesterId := uuid.NewString()
			request.SetRequesterId(requesterId)
			response, httpResponse, err := c.StageAPI.SuperplaneCreateStage(context.Background(), canvasIDOrName).
				Body(*request).
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
