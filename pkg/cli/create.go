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

			domainId, ok := metadata["domainId"].(string)
			if !ok {
				Fail("Invalid Secret YAML: domainId field missing")
			}

			domainTypeFromYaml, ok := metadata["domainType"].(string)
			if !ok {
				Fail("Invalid Secret YAML: domainType field missing")
			}

			var secret openapi_client.SecretsSecret
			err = yaml.Unmarshal(data, &secret)
			Check(err)

			domainType, err := openapi_client.NewAuthorizationDomainTypeFromValue(domainTypeFromYaml)
			Check(err)

			response, httpResponse, err := c.SecretAPI.
				SecretsCreateSecret(context.Background()).
				Body(openapi_client.SecretsCreateSecretRequest{
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

			canvasIDOrName, ok := metadata["canvasId"].(string)
			if !ok {
				canvasIDOrName, ok = metadata["canvasName"].(string)
				if !ok {
					Fail("Invalid EventSource YAML: canvasId or canvasName field missing")
				}
			}

			// Create the event source request
			var eventSource openapi_client.SuperplaneEventSource
			err = yaml.Unmarshal(data, &eventSource)
			Check(err)

			body := openapi_client.SuperplaneCreateEventSourceBody{
				EventSource: &eventSource,
			}

			response, httpResponse, err := c.EventSourceAPI.SuperplaneCreateEventSource(context.Background(), canvasIDOrName).Body(body).Execute()
			if err != nil {
				b, _ := io.ReadAll(httpResponse.Body)
				fmt.Printf("%s\n", string(b))
				os.Exit(1)
			}

			// Access the event source from response
			es := response.GetEventSource()
			fmt.Printf("Event Source '%s' created with ID '%s'.\n",
				*es.GetMetadata().Name, *es.GetMetadata().Id)
			fmt.Printf("Key: %s\n", *response.Key)
			fmt.Println("! Save this key as it won't be shown again.")

		case "ConnectionGroup":
			// Parse YAML to map
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			// Extract the metadata from the YAML
			metadata, ok := yamlData["metadata"].(map[string]interface{})
			if !ok {
				Fail("Invalid ConnectionGroup YAML: metadata section missing")
			}

			canvasID, ok := metadata["canvasId"].(string)
			if !ok {
				Fail("Invalid ConnectionGroup YAML: canvasId or canvasName field missing")
			}

			var connectionGroup openapi_client.SuperplaneConnectionGroup
			err = yaml.Unmarshal(data, &connectionGroup)
			Check(err)

			body := openapi_client.SuperplaneCreateConnectionGroupBody{
				ConnectionGroup: &connectionGroup,
			}

			response, httpResponse, err := c.ConnectionGroupAPI.SuperplaneCreateConnectionGroup(context.Background(), canvasID).
				Body(body).
				Execute()

			if err != nil {
				b, _ := io.ReadAll(httpResponse.Body)
				fmt.Printf("%s\n", string(b))
				os.Exit(1)
			}

			out, err := yaml.Marshal(response.ConnectionGroup)
			Check(err)
			fmt.Printf("%s", string(out))

		case "Integration":
			// Parse YAML to map
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			// Extract the metadata from the YAML
			metadata, ok := yamlData["metadata"].(map[string]interface{})
			if !ok {
				Fail("Invalid Integration YAML: metadata section missing")
			}

			domainId, ok := metadata["domainId"].(string)
			if !ok {
				Fail("Invalid Integration YAML: domainId field missing")
			}

			domainTypeFromYaml, ok := metadata["domainType"].(string)
			if !ok {
				Fail("Invalid Integration YAML: domainType field missing")
			}

			var integration openapi_client.IntegrationsIntegration
			err = yaml.Unmarshal(data, &integration)
			Check(err)

			domainType, err := openapi_client.NewAuthorizationDomainTypeFromValue(domainTypeFromYaml)
			Check(err)

			response, httpResponse, err := c.IntegrationAPI.
				IntegrationsCreateIntegration(context.Background()).
				Body(openapi_client.IntegrationsCreateIntegrationRequest{
					Integration: &integration,
					DomainId:    &domainId,
					DomainType:  domainType,
				}).
				Execute()

			if err != nil {
				b, _ := io.ReadAll(httpResponse.Body)
				fmt.Printf("%s\n", string(b))
				os.Exit(1)
			}

			out, err := yaml.Marshal(response.Integration)
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
