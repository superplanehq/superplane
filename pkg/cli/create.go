package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/models"
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
		case models.CanvasKind:
			resource, err := models.ParseCanvas(data)
			Check(err)

			canvas := models.CanvasFromCanvas(*resource)
			request := openapi_client.CanvasesCreateCanvasRequest{}
			request.SetCanvas(canvas)

			client := DefaultClient()
			_, _, err = client.CanvasAPI.CanvasesCreateCanvas(context.Background()).Body(request).Execute()
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

		resource := models.Canvas{
			APIVersion: APIVersion,
			Kind:       models.CanvasKind,
			Metadata:   &openapi_client.CanvasesCanvasMetadata{Name: &name},
			Spec:       models.EmptyCanvasSpec(),
		}

		canvas := models.CanvasFromCanvas(resource)
		request := openapi_client.CanvasesCreateCanvasRequest{}
		request.SetCanvas(canvas)

		_, _, err := client.CanvasAPI.CanvasesCreateCanvas(context.Background()).Body(request).Execute()
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
