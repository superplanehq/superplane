package memory

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "memory",
		Short: "Manage app memory",
		Long:  "List memory records stored by an app.",
	}

	listCmd := &cobra.Command{
		Use:   "list [app-name-or-id]",
		Short: "List app memory records",
		Long: `List memory records stored by an app.

Use --namespace to show records from one namespace.
The app argument is optional. When omitted, the active app
configured with "superplane apps active" is used.`,
		Args: cobra.MaximumNArgs(1),
	}
	var listNamespace string
	listCmd.Flags().StringVar(&listNamespace, "namespace", "", "filter memory records by namespace")
	core.Bind(listCmd, &listCommand{namespace: &listNamespace}, options)

	root.AddCommand(listCmd)

	return root
}
