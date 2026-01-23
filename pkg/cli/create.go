package cli

import (
	"context"
	"fmt"
	"os"

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

		switch kind {
		case canvasKind:
			resource, err := ParseCanvasResource(data)
			Check(err)

			workflow := WorkflowFromCanvasResource(*resource)
			request := openapi_client.WorkflowsCreateWorkflowRequest{}
			request.SetWorkflow(workflow)

			client := DefaultClient()
			_, _, err = client.WorkflowAPI.WorkflowsCreateWorkflow(context.Background()).Body(request).Execute()
			Check(err)
		default:
			Fail(fmt.Sprintf("Unsupported resource kind '%s'", kind))
		}
	},
}

var createCanvasCmd = &cobra.Command{
	Use:   "canvas <canvas-name>",
	Short: "Create a canvas",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := DefaultClient()

		resource := CanvasResource{
			APIVersion: canvasAPIVersion,
			Kind:       canvasKind,
			Metadata:   &openapi_client.WorkflowsWorkflowMetadata{Name: &name},
			Spec:       EmptyWorkflowSpec(),
		}

		workflow := WorkflowFromCanvasResource(resource)
		request := openapi_client.WorkflowsCreateWorkflowRequest{}
		request.SetWorkflow(workflow)

		_, _, err := client.WorkflowAPI.WorkflowsCreateWorkflow(context.Background()).Body(request).Execute()
		Check(err)
	},
}

func init() {
	RootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createCanvasCmd)

	// File flag for root create command
	desc := "Filename, directory, or URL to files to use to create the resource"
	createCmd.Flags().StringP("file", "f", "", desc)
}
