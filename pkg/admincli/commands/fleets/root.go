package fleets

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "fleets",
		Short: "Manage runner fleets",
	}

	root.AddCommand(newListCommand(options))
	root.AddCommand(newDescribeCommand(options))
	root.AddCommand(newCreateCommand(options))
	root.AddCommand(newUpdateCommand(options))
	return root
}
