package drafts

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "drafts",
		Short: "Manage app draft git branches",
		Long:  "Create, list, and delete draft git branches used for in-progress app edits.",
	}

	createCmd := &cobra.Command{
		Use:   "create [app-name-or-id]",
		Short: "Create a draft branch",
		Long: `Create a draft git branch for editing an app. When omitted, the branch name
defaults to drafts/<your-user-id>. Re-running create is idempotent for the default branch.`,
		Args: cobra.MaximumNArgs(1),
	}
	var displayName string
	createCmd.Flags().StringVar(&displayName, "name", "", "optional display name for the draft branch")
	core.Bind(createCmd, &createCommand{displayName: &displayName}, options)

	listCmd := &cobra.Command{
		Use:   "list [app-name-or-id]",
		Short: "List draft branches",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(listCmd, &listCommand{}, options)

	deleteCmd := &cobra.Command{
		Use:   "delete <branch-name> [app-name-or-id]",
		Short: "Delete a draft branch",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(deleteCmd, &deleteCommand{}, options)

	root.AddCommand(createCmd)
	root.AddCommand(listCmd)
	root.AddCommand(deleteCmd)
	return root
}
