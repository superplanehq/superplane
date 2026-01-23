package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

// Root describe command
var getCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show details of SuperPlane resources",
	Long:    `Get detailed information about SuperPlane resources.`,
	Aliases: []string{"desc", "get"},
}

var getCanvasCmd = &cobra.Command{
	Use:   "canvas <canvas-name>",
	Short: "Get a canvas",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := DefaultClient()
		ctx := context.Background()

		workflowID, err := findWorkflowIDByName(ctx, client, name)
		Check(err)

		response, _, err := client.WorkflowAPI.WorkflowsDescribeWorkflow(ctx, workflowID).Execute()
		Check(err)

		if response.Workflow == nil {
			Fail(fmt.Sprintf("canvas %q not found", name))
		}

		resource := CanvasResourceFromWorkflow(*response.Workflow)
		output, err := yaml.Marshal(resource)
		Check(err)

		fmt.Fprintln(os.Stdout, string(output))
	},
}

func init() {
	RootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getCanvasCmd)
}
