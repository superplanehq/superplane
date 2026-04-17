package groups

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "groups",
		Short:   "Manage organization groups",
		Aliases: []string{"group"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List groups",
		Args:  cobra.NoArgs,
	}
	core.Bind(listCmd, &listCommand{}, options)

	getCmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get a group",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	var createFile string
	var createDisplayName string
	var createDescription string
	var createRole string
	createCmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a group from inline flags or a file",
		Long:  "Create a group by passing a name positional with --display-name / --description / --role, or by passing --file pointing to a YAML resource definition.",
		Args:  cobra.MaximumNArgs(1),
	}
	createCmd.Flags().StringVarP(&createFile, "file", "f", "", "path to a YAML file describing the group")
	createCmd.Flags().StringVar(&createDisplayName, "display-name", "", "group display name")
	createCmd.Flags().StringVar(&createDescription, "description", "", "group description")
	createCmd.Flags().StringVar(&createRole, "role", "", "role assigned to members of the group")
	core.Bind(createCmd, &createCommand{
		file:        &createFile,
		displayName: &createDisplayName,
		description: &createDescription,
		role:        &createRole,
	}, options)

	var updateFile string
	var updateDisplayName string
	var updateDescription string
	var updateRole string
	updateCmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Update a group inline or from a file",
		Args:  cobra.MaximumNArgs(1),
	}
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "path to a YAML file describing the group")
	updateCmd.Flags().StringVar(&updateDisplayName, "display-name", "", "group display name")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "group description")
	updateCmd.Flags().StringVar(&updateRole, "role", "", "role assigned to members of the group")
	core.Bind(updateCmd, &updateCommand{
		file:        &updateFile,
		displayName: &updateDisplayName,
		description: &updateDescription,
		role:        &updateRole,
	}, options)

	deleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a group",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(deleteCmd, &deleteCommand{}, options)

	membersCmd := &cobra.Command{
		Use:     "members",
		Short:   "Manage members of a group",
		Aliases: []string{"member"},
	}

	membersListCmd := &cobra.Command{
		Use:   "list <group>",
		Short: "List members of a group",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(membersListCmd, &membersListCommand{}, options)

	var membersAddEmail string
	membersAddCmd := &cobra.Command{
		Use:   "add <group> [user-id]",
		Short: "Add a member to a group",
		Args:  cobra.RangeArgs(1, 2),
	}
	membersAddCmd.Flags().StringVar(&membersAddEmail, "email", "", "identify user by email instead of id")
	core.Bind(membersAddCmd, &membersAddCommand{email: &membersAddEmail}, options)

	var membersRemoveEmail string
	membersRemoveCmd := &cobra.Command{
		Use:   "remove <group> [user-id]",
		Short: "Remove a member from a group",
		Args:  cobra.RangeArgs(1, 2),
	}
	membersRemoveCmd.Flags().StringVar(&membersRemoveEmail, "email", "", "identify user by email instead of id")
	core.Bind(membersRemoveCmd, &membersRemoveCommand{email: &membersRemoveEmail}, options)

	membersCmd.AddCommand(membersListCmd)
	membersCmd.AddCommand(membersAddCmd)
	membersCmd.AddCommand(membersRemoveCmd)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)
	root.AddCommand(createCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(deleteCmd)
	root.AddCommand(membersCmd)

	return root
}
