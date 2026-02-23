package plugins

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	root := &cobra.Command{
		Use:     "plugin",
		Short:   "Manage SuperPlane plugins",
		Aliases: []string{"plugins"},
	}

	root.AddCommand(newInstallCommand())
	root.AddCommand(newUninstallCommand())
	root.AddCommand(newListCommand())
	root.AddCommand(newPackCommand())

	return root
}
