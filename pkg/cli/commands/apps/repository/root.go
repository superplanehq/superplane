package repository

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewRootCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "repository",
		Short: "App git repository metadata",
		Long:  "Inspect the git repository attached to an app.",
		Aliases: []string{
			"repo",
		},
	}

	getCmd := &cobra.Command{
		Use:   "get [app-name-or-id]",
		Short: "Show app repository metadata",
		Long: `Print metadata for the git repository attached to an app, including
repository id, storage provider, URL, default branch, state, and head SHA.

The app argument is optional. When omitted, the active app configured with
"superplane apps active" is used.`,
		Args: cobra.MaximumNArgs(1),
	}

	core.Bind(getCmd, &GetCommand{}, options)
	root.AddCommand(getCmd)

	return root
}
