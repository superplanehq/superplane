package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// Root list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List SuperPlane resources",
	Long:    `List multiple SuperPlane resources.`,
	Aliases: []string{"ls"},
}

var listCanvasCmd = &cobra.Command{
	Use:     "canvas",
	Short:   "List canvases",
	Aliases: []string{"canvases"},
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := DefaultClient()
		ctx := context.Background()
		response, _, err := client.WorkflowAPI.WorkflowsListWorkflows(ctx).Execute()
		Check(err)

		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(writer, "ID\tNAME\tCREATED_AT")
		for _, workflow := range response.Workflows {
			metadata := workflow.GetMetadata()
			createdAt := ""
			if metadata.HasCreatedAt() {
				createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\n", metadata.GetId(), metadata.GetName(), createdAt)
		}
		_ = writer.Flush()
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listCanvasCmd)
}
