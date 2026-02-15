package triggers

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "triggers",
		Short: "Manage triggers",
	}

	var from string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List triggers",
		Args:  cobra.NoArgs,
	}
	listCmd.Flags().StringVar(&from, "from", "", "integration name")
	core.Bind(listCmd, &listCommand{from: &from}, options)

	getCmd := &cobra.Command{
		Use:   "get <trigger-name>",
		Short: "Get a trigger",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)

	return root
}
