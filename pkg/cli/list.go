package cli

import (
	"github.com/spf13/cobra"
)

// Root list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List SuperPlane resources",
	Long:    `List multiple SuperPlane resources.`,
	Aliases: []string{"ls"},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
