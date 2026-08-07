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

	appIDUsage := "scope the command to this app/canvas instead of the organization"

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List secrets",
		Args:  cobra.NoArgs,
	}
	var listAppID string
	core.BindAppIDFlag(listCmd, &listAppID, appIDUsage)
	core.Bind(listCmd, &listCommand{appID: &listAppID}, options)

	getCmd := &cobra.Command{
		Use:   "get <id-or-name>",
		Short: "Get a secret",
		Args:  cobra.ExactArgs(1),
	}
	var getAppID string
	core.BindAppIDFlag(getCmd, &getAppID, appIDUsage)
	core.Bind(getCmd, &getCommand{appID: &getAppID}, options)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a secret",
		Long:  core.AgentSkillsHelp(),
		Args:  cobra.NoArgs,
	}
	var createFile string
	var createAppID string
	createCmd.Flags().StringVarP(&createFile, "file", "f", "", "path to resource file, or - to read from stdin")
	_ = createCmd.MarkFlagRequired("file")
	core.BindAppIDFlag(createCmd, &createAppID, appIDUsage)
	core.Bind(createCmd, &createCommand{file: &createFile, appID: &createAppID}, options)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a secret from a file",
		Args:  cobra.NoArgs,
	}
	var updateFile string
	var updateAppID string
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "path to resource file, or - to read from stdin")
	_ = updateCmd.MarkFlagRequired("file")
	core.BindAppIDFlag(updateCmd, &updateAppID, appIDUsage)
	core.Bind(updateCmd, &updateCommand{file: &updateFile, appID: &updateAppID}, options)

	deleteCmd := &cobra.Command{
		Use:   "delete <id-or-name>",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
	}
	var deleteAppID string
	core.BindAppIDFlag(deleteCmd, &deleteAppID, appIDUsage)
	core.Bind(deleteCmd, &deleteCommand{appID: &deleteAppID}, options)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)
	root.AddCommand(createCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(deleteCmd)

	return root
}
