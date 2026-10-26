package members

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "members",
		Short:   "Manage organization members",
		Aliases: []string{"member"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List organization members",
		Args:  cobra.NoArgs,
	}
	core.Bind(listCmd, &listCommand{}, options)

	var getEmail string
	getCmd := &cobra.Command{
		Use:   "get [user-id]",
		Short: "Get an organization member",
		Long:  "Get details of a member by user id or --email. The member list is filtered client-side since the API has no single-member lookup.",
		Args:  cobra.MaximumNArgs(1),
	}
	getCmd.Flags().StringVar(&getEmail, "email", "", "lookup member by email instead of id")
	core.Bind(getCmd, &getCommand{email: &getEmail}, options)

	var updateEmail string
	var updateRole string
	updateCmd := &cobra.Command{
		Use:   "update [user-id]",
		Short: "Update a member's role",
		Args:  cobra.MaximumNArgs(1),
	}
	updateCmd.Flags().StringVar(&updateEmail, "email", "", "identify member by email instead of id")
	updateCmd.Flags().StringVar(&updateRole, "role", "", "role name to assign to the member")
	_ = updateCmd.MarkFlagRequired("role")
	core.Bind(updateCmd, &updateCommand{email: &updateEmail, role: &updateRole}, options)

	var removeEmail string
	removeCmd := &cobra.Command{
		Use:   "remove [user-id]",
		Short: "Remove a member from the organization",
		Args:  cobra.MaximumNArgs(1),
	}
	removeCmd.Flags().StringVar(&removeEmail, "email", "", "identify member by email instead of id")
	core.Bind(removeCmd, &removeCommand{email: &removeEmail}, options)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(removeCmd)

	return root
}
