package secrets

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "secrets",
		Short:   "Manage secrets",
		Aliases: []string{"secret"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List secrets",
		Args:  cobra.NoArgs,
	}
	core.Bind(listCmd, &listCommand{}, options)

	getCmd := &cobra.Command{
		Use:   "get <id-or-name>",
		Short: "Get a secret",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a secret",
		Args:  cobra.NoArgs,
	}
	var createFile string
	createCmd.Flags().StringVarP(&createFile, "file", "f", "", "filename, directory, or URL to files to use to create the resource")
	_ = createCmd.MarkFlagRequired("file")
	core.Bind(createCmd, &createCommand{file: &createFile}, options)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a secret from a file",
		Args:  cobra.NoArgs,
	}
	var updateFile string
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "filename, directory, or URL to files to use to update the resource")
	_ = updateCmd.MarkFlagRequired("file")
	core.Bind(updateCmd, &updateCommand{file: &updateFile}, options)

	deleteCmd := &cobra.Command{
		Use:   "delete <id-or-name>",
		Short: "Delete a secret",
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
