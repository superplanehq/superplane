package drafts

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "drafts",
		Short:   "Manage app drafts",
		Aliases: []string{"draft"},
		Long: `Create, list, and delete draft versions for an app.

Drafts are in-progress app versions (canvas.yaml and console.yaml) owned by a user.
Use "superplane apps drafts list" to find draft ids, then pass --draft-id to canvas
and console commands.`,
	}

	var createName string
	createCmd := &cobra.Command{
		Use:   "create [app]",
		Short: "Create a new draft for an app",
		Args:  cobra.MaximumNArgs(1),
	}
	createCmd.Flags().StringVar(&createName, "name", "", "display name for the draft")
	core.Bind(createCmd, &createCommand{name: &createName}, options)

	var listAll bool
	listCmd := &cobra.Command{
		Use:   "list [app]",
		Short: "List drafts for an app",
		Args:  cobra.MaximumNArgs(1),
	}
	listCmd.Flags().BoolVar(&listAll, "all", false, "list drafts for all owners (best-effort)")
	core.Bind(listCmd, &listCommand{all: &listAll}, options)

	deleteCmd := &cobra.Command{
		Use:   "delete <draft-id> [app]",
		Short: "Delete a draft",
		Long: `Delete a draft.

The app is taken from the optional [app] argument, falling back to the active app
configured with "superplane apps active".`,
		Args: cobra.RangeArgs(1, 2),
	}
	core.Bind(deleteCmd, &deleteCommand{}, options)

	root.AddCommand(createCmd)
	root.AddCommand(listCmd)
	root.AddCommand(deleteCmd)

	return root
}
