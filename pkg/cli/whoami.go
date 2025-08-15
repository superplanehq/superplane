package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	Short:   "Get information about the currently authenticated user",
	Aliases: []string{"events"},
	Args:    cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		c := DefaultClient()

		response, _, err := c.MeAPI.MeMe(context.Background()).Execute()
		Check(err)

		fmt.Printf("ID: %s\n", response.GetId())
		fmt.Printf("Email: %s\n", response.GetEmail())
		fmt.Printf("Organization: %s\n", response.GetOrganizationId())
	},
}

func init() {
	RootCmd.AddCommand(whoamiCmd)
}
