package apps

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "apps",
		Short:   "Manage SuperPlane Apps",
		Aliases: []string{"app"},
	}

	// list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List apps in the current organization",
		Args:  cobra.NoArgs,
	}
	core.Bind(listCmd, &listCommand{}, options)

	// describe
	describeCmd := &cobra.Command{
		Use:   "describe <name-or-id>",
		Short: "Show app details and sync state",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(describeCmd, &describeCommand{}, options)

	// create
	var createDisplayName string
	var createAppSlug string
	var createDescription string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new app",
		Args:  cobra.NoArgs,
	}
	createCmd.Flags().StringVar(&createDisplayName, "display-name", "", "human-readable display name for the app (required)")
	createCmd.Flags().StringVar(&createAppSlug, "app-slug", "", "slug segment for the app (lowercase letters, digits, underscores; required)")
	createCmd.Flags().StringVar(&createDescription, "description", "", "optional description for the app")
	_ = createCmd.MarkFlagRequired("display-name")
	_ = createCmd.MarkFlagRequired("app-slug")
	core.Bind(createCmd, &createCommand{
		displayName: &createDisplayName,
		appSlug:     &createAppSlug,
		description: &createDescription,
	}, options)

	// delete
	var deleteYes bool
	deleteCmd := &cobra.Command{
		Use:   "delete <name-or-id>",
		Short: "Delete an app (prompts for confirmation unless --yes is passed)",
		Args:  cobra.ExactArgs(1),
	}
	deleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "skip confirmation prompt")
	core.Bind(deleteCmd, &deleteCommand{yes: &deleteYes}, options)

	// sync
	syncCmd := &cobra.Command{
		Use:   "sync <name-or-id>",
		Short: "Trigger a manual sync from Code Storage",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(syncCmd, &syncCommand{}, options)

	root.AddCommand(listCmd)
	root.AddCommand(describeCmd)
	root.AddCommand(createCmd)
	root.AddCommand(deleteCmd)
	root.AddCommand(syncCmd)

	return root
}
