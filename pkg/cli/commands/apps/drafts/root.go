package drafts

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "drafts",
		Short:   "Manage staged app edits",
		Aliases: []string{"draft"},
		Long: `Inspect and discard per-user staged edits for an app.

Use "superplane apps drafts list" to see staged paths, then commit from the UI
or with canvas update without --draft-id.`,
	}

	var createName string
	createCmd := &cobra.Command{
		Use:   "create [app]",
		Short: "Show the live version used for staging",
		Args:  cobra.MaximumNArgs(1),
	}
	createCmd.Flags().StringVar(&createName, "name", "", "ignored; kept for compatibility")
	core.Bind(createCmd, &createCommand{name: &createName}, options)

	listCmd := &cobra.Command{
		Use:   "list [app]",
		Short: "List staged changes for an app",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(listCmd, &listCommand{}, options)

	deleteCmd := &cobra.Command{
		Use:   "delete [app]",
		Short: "Discard staged changes for an app",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(deleteCmd, &deleteCommand{}, options)

	root.AddCommand(createCmd)
	root.AddCommand(listCmd)
	root.AddCommand(deleteCmd)

	return root
}
