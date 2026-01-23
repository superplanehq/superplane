package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a resource from a file.",
	Long:    `Update a SuperPlane resource from a YAML file.`,
	Aliases: []string{"update", "edit"},

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

			client := DefaultClient()
			ctx := context.Background()

			workflowID := *resource.Metadata.Id
			if workflowID == "" {
				workflowID, err = findWorkflowIDByName(ctx, client, *resource.Metadata.Name)
				Check(err)
			}

			workflow := WorkflowFromCanvasResource(*resource)
			body := openapi_client.WorkflowsUpdateWorkflowBody{}
			body.SetWorkflow(workflow)

			_, _, err = client.WorkflowAPI.WorkflowsUpdateWorkflow(ctx, workflowID).Body(body).Execute()
			Check(err)
		default:
			Fail(fmt.Sprintf("Unsupported resource kind '%s' for update", kind))
		}
	},
}

func init() {
	RootCmd.AddCommand(updateCmd)

	// File flag for root update command
	desc := "Filename, directory, or URL to files to use to update the resource"
	updateCmd.Flags().StringP("file", "f", "", desc)
}
