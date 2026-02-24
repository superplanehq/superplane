package plugins

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "plugin",
		Short:   "Manage SuperPlane plugins",
		Aliases: []string{"plugins"},
	}

	root.AddCommand(newInstallCommand(options))
	root.AddCommand(newUninstallCommand(options))
	root.AddCommand(newListCommand())
	root.AddCommand(newPackCommand())

	return root
}
