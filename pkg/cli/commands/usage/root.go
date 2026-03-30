package usage

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "usage",
		Short: "Inspect organization usage and plan limits",
	}

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get usage details for the current organization",
		Args:  cobra.NoArgs,
	}
	core.Bind(getCmd, &getCommand{}, options)

	root.AddCommand(getCmd)

	return root
}
