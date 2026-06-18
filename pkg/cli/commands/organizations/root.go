package organizations

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "organizations",
		Short:   "Manage organizations",
		Aliases: []string{"org", "orgs"},
	}

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get details for the current organization",
		Args:  cobra.NoArgs,
	}
	core.Bind(getCmd, &getCommand{}, options)

	var updateName string
	var updateDescription string
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update the current organization",
		Args:  cobra.NoArgs,
	}
	updateCmd.Flags().StringVar(&updateName, "name", "", "organization name")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "organization description")
	core.Bind(updateCmd, &updateCommand{
		name:        &updateName,
		description: &updateDescription,
	}, options)

	root.AddCommand(getCmd)
	root.AddCommand(updateCmd)

	return root
}
