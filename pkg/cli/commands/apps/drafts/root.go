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
		Use:   "delete <draft-id>",
		Short: "Delete a draft",
		Long: `Delete a draft.

The app is resolved from the active app, or by searching the drafts of every app
you can access when no active app is set.`,
		Args: cobra.ExactArgs(1),
	}
	core.Bind(deleteCmd, &deleteCommand{}, options)

	stagingCmd := &cobra.Command{
		Use:   "staging",
		Short: "Commit or reset draft staging",
		Long: `Commit staged canvas.yaml/console.yaml into the draft version row, or
discard uncommitted staging without touching the committed draft.`,
	}

	stagingCommitCmd := &cobra.Command{
		Use:   "commit <draft-id> [app]",
		Short: "Commit staged changes into the draft version",
		Long: `Parse staged canvas.yaml and console.yaml into the draft version row and
clear staging. Requires canvases:update permission (not available to agent tokens).`,
		Args: cobra.RangeArgs(1, 2),
	}
	core.Bind(stagingCommitCmd, &stagingCommitCommand{}, options)

	var resetPaths []string
	stagingResetCmd := &cobra.Command{
		Use:   "reset <draft-id> [app]",
		Short: "Discard staged changes",
		Long: `Discard uncommitted staging for a draft. Pass --path to revert a single
spec file, or omit paths to discard all staged edits.`,
		Args: cobra.RangeArgs(1, 2),
	}
	stagingResetCmd.Flags().StringArrayVar(&resetPaths, "path", nil, "spec file to revert (canvas.yaml or console.yaml; repeatable)")
	core.Bind(stagingResetCmd, &stagingResetCommand{paths: &resetPaths}, options)

	stagingCmd.AddCommand(stagingCommitCmd)
	stagingCmd.AddCommand(stagingResetCmd)

	root.AddCommand(createCmd)
	root.AddCommand(listCmd)
	root.AddCommand(deleteCmd)
	root.AddCommand(stagingCmd)

	return root
}
