package cli

import (
	"github.com/spf13/cobra"
)

// Root describe command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete SuperPlane resources",
	Long:  `Delete a SuperPlane resource by ID or name.`,
}

func init() {
	RootCmd.AddCommand(deleteCmd)
}
