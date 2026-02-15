package components

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "components",
		Short: "Manage components",
	}

	var from string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List components",
		Args:  cobra.NoArgs,
	}
	listCmd.Flags().StringVar(&from, "from", "", "integration name")
	core.Bind(listCmd, &listCommand{from: &from}, options)

	getCmd := &cobra.Command{
		Use:   "get <component-name>",
		Short: "Get a component",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)

	return root
}
