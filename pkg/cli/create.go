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
	Long:  `Create a SuperPlane resource from a YAML file.`,

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
			// Parse YAML to workflow structure
			var yamlData map[string]any
			err = yaml.Unmarshal(data, &yamlData)
			Check(err)

			// Extract metadata
			metadata, ok := yamlData["metadata"].(map[string]interface{})
			if !ok {
				Fail("Invalid Canvas YAML: metadata section missing")
			}

			name, ok := metadata["name"].(string)
			if !ok || name == "" {
				Fail("Invalid Canvas YAML: name is required in metadata")
			}

			description := ""
			if desc, ok := metadata["description"].(string); ok {
				description = desc
			}

			// Extract spec (optional for empty workflows)
			var nodes []openapi_client.ComponentsNode
			var edges []openapi_client.ComponentsEdge

			if spec, ok := yamlData["spec"].(map[string]interface{}); ok {
				// Parse nodes
				if nodesData, ok := spec["nodes"].([]interface{}); ok {
					nodesBytes, err := yaml.Marshal(nodesData)
					Check(err)
					err = yaml.Unmarshal(nodesBytes, &nodes)
					Check(err)
				}

				// Parse edges
				if edgesData, ok := spec["edges"].([]interface{}); ok {
					edgesBytes, err := yaml.Marshal(edgesData)
					Check(err)
					err = yaml.Unmarshal(edgesBytes, &edges)
					Check(err)
				}
			}

			// Create workflow request
			workflowRequest := openapi_client.WorkflowsWorkflow{
				Metadata: &openapi_client.WorkflowsWorkflowMetadata{
					Name:        &name,
					Description: &description,
				},
				Spec: &openapi_client.WorkflowsWorkflowSpec{
					Nodes: nodes,
					Edges: edges,
				},
			}

			response, httpResponse, err := c.WorkflowAPI.
				WorkflowsCreateWorkflow(context.Background()).
				Body(openapi_client.WorkflowsCreateWorkflowRequest{
					Workflow: &workflowRequest,
				}).
				Execute()

			if err != nil {
				b, _ := io.ReadAll(httpResponse.Body)
				fmt.Printf("%s\n", string(b))
				os.Exit(1)
			}

			out, err := yaml.Marshal(response.Workflow)
			Check(err)
			fmt.Printf("Canvas created successfully:\n%s", string(out))

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
