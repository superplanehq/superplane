package roles

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "roles",
		Short:   "Manage organization roles",
		Aliases: []string{"role"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List roles",
		Args:  cobra.NoArgs,
	}
	core.Bind(listCmd, &listCommand{}, options)

	getCmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get a role",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	var createFile string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a role from a file",
		Args:  cobra.NoArgs,
	}
	createCmd.Flags().StringVarP(&createFile, "file", "f", "", "filename, directory, or URL to files to use to create the resource")
	_ = createCmd.MarkFlagRequired("file")
	core.Bind(createCmd, &createCommand{file: &createFile}, options)

	var updateFile string
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a role from a file",
		Args:  cobra.NoArgs,
	}
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "filename, directory, or URL to files to use to update the resource")
	_ = updateCmd.MarkFlagRequired("file")
	core.Bind(updateCmd, &updateCommand{file: &updateFile}, options)

	deleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a role",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(deleteCmd, &deleteCommand{}, options)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)
	root.AddCommand(createCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(deleteCmd)

	return root
}
