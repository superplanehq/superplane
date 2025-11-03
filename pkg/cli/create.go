package cli

import (
	"context"
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
