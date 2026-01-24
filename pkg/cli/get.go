package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/models"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// Root describe command
var getCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show details of SuperPlane resources",
	Long:    `Get detailed information about SuperPlane resources.`,
	Aliases: []string{"desc", "get"},
}

var getCanvasCmd = &cobra.Command{
	Use:   "canvas <name-or-id>",
	Short: "Get a canvas",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nameOrID := args[0]
		client := DefaultClient()
		ctx := context.Background()

		workflowID, err := findWorkflowID(ctx, client, nameOrID)
		Check(err)

		response, _, err := client.WorkflowAPI.WorkflowsDescribeWorkflow(ctx, workflowID).Execute()
		Check(err)

		resource := models.CanvasResourceFromWorkflow(*response.Workflow)
		output, err := yaml.Marshal(resource)
		Check(err)

		fmt.Fprintln(os.Stdout, string(output))
	},
}

func findWorkflowID(ctx context.Context, client *openapi_client.APIClient, nameOrID string) (string, error) {
	_, err := uuid.Parse(nameOrID)
	if err == nil {
		return nameOrID, nil
	}

	return findWorkflowIDByName(ctx, client, nameOrID)
}

func init() {
	RootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getCanvasCmd)
}
