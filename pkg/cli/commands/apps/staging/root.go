package staging

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "staging",
		Short: "Manage staged app edits",
		Long: `Stage, inspect, and commit uncommitted app changes.

Use "superplane apps staging list" to see staged paths, "staging update" to
stage files, and "staging commit" to publish staged edits.

To commit a single file directly without a separate staging step, pass
--message to "superplane apps canvas update" or "superplane apps console set".`,
	}

	listCmd := &cobra.Command{
		Use:   "list [app]",
		Short: "List staged changes for an app",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(listCmd, &listCommand{}, options)

	var updateFiles []string
	updateCmd := &cobra.Command{
		Use:   "update [app]",
		Short: "Stage one or more repository files",
		Long: `Stage local files for an app without committing them.

Each --file path is mapped to a repository path using the file name only
(for example, canvas.yaml and README.md).`,
		Args: cobra.MaximumNArgs(1),
	}
	updateCmd.Flags().StringArrayVar(&updateFiles, "file", nil, "local file to stage (repeatable)")
	_ = updateCmd.MarkFlagRequired("file")
	core.Bind(updateCmd, &updateCommand{files: &updateFiles}, options)

	var commitMessage string
	commitCmd := &cobra.Command{
		Use:   "commit [app]",
		Short: "Commit staged changes",
		Args:  cobra.MaximumNArgs(1),
	}
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "commit message")
	_ = commitCmd.MarkFlagRequired("message")
	core.Bind(commitCmd, &commitCommand{message: &commitMessage}, options)

	root.AddCommand(listCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(commitCmd)

	return root
}
