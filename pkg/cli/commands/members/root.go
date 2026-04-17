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

	invitationsCmd := &cobra.Command{
		Use:     "invitations",
		Short:   "Manage pending organization invitations",
		Aliases: []string{"invitation"},
	}

	invitationsListCmd := &cobra.Command{
		Use:   "list",
		Short: "List pending invitations",
		Args:  cobra.NoArgs,
	}
	core.Bind(invitationsListCmd, &invitationsListCommand{}, options)

	var invitationsCreateEmail string
	invitationsCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an invitation",
		Args:  cobra.NoArgs,
	}
	invitationsCreateCmd.Flags().StringVar(&invitationsCreateEmail, "email", "", "email address to invite")
	_ = invitationsCreateCmd.MarkFlagRequired("email")
	core.Bind(invitationsCreateCmd, &invitationsCreateCommand{email: &invitationsCreateEmail}, options)

	invitationsRemoveCmd := &cobra.Command{
		Use:   "remove <invitation-id>",
		Short: "Remove a pending invitation",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(invitationsRemoveCmd, &invitationsRemoveCommand{}, options)

	invitationsCmd.AddCommand(invitationsListCmd)
	invitationsCmd.AddCommand(invitationsCreateCmd)
	invitationsCmd.AddCommand(invitationsRemoveCmd)

	inviteLinkCmd := &cobra.Command{
		Use:   "invite-link",
		Short: "Manage the organization invite link",
	}

	inviteLinkGetCmd := &cobra.Command{
		Use:   "get",
		Short: "Get the organization invite link",
		Args:  cobra.NoArgs,
	}
	core.Bind(inviteLinkGetCmd, &inviteLinkGetCommand{}, options)

	var inviteLinkEnabled bool
	inviteLinkUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Enable or disable the invite link",
		Args:  cobra.NoArgs,
	}
	inviteLinkUpdateCmd.Flags().BoolVar(&inviteLinkEnabled, "enabled", false, "whether the invite link is enabled")
	_ = inviteLinkUpdateCmd.MarkFlagRequired("enabled")
	core.Bind(inviteLinkUpdateCmd, &inviteLinkUpdateCommand{enabled: &inviteLinkEnabled}, options)

	inviteLinkResetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Rotate the invite link token",
		Args:  cobra.NoArgs,
	}
	core.Bind(inviteLinkResetCmd, &inviteLinkResetCommand{}, options)

	inviteLinkCmd.AddCommand(inviteLinkGetCmd)
	inviteLinkCmd.AddCommand(inviteLinkUpdateCmd)
	inviteLinkCmd.AddCommand(inviteLinkResetCmd)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(removeCmd)
	root.AddCommand(invitationsCmd)
	root.AddCommand(inviteLinkCmd)

	return root
}
