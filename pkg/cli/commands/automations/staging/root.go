package staging

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:    "staging",
		Short:  "Manage staged automation edits",
		Hidden: true,
		Long: `Stage, inspect, and commit uncommitted automation changes.

Use "superplane automations staging status" to see staged paths, "staging update" to
stage files, and "staging commit" to publish staged edits.

To commit a file directly without a separate staging step, pass
--message to "superplane automations update".`,
	}

	statusCmd := &cobra.Command{
		Use:   "status [automation]",
		Short: "Show staged changes for an automation",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(statusCmd, &statusCommand{}, options)

	var updateFiles []string
	updateCmd := &cobra.Command{
		Use:   "update [automation]",
		Short: "Stage canvas.yaml",
		Long: `Stage a local automation definition without committing it.

Each --file path is mapped to a repository path using the file name only.
Only canvas.yaml can be staged.`,
		Args: cobra.MaximumNArgs(1),
	}
	updateCmd.Flags().StringArrayVar(&updateFiles, "file", nil, "local file to stage (repeatable)")
	_ = updateCmd.MarkFlagRequired("file")
	core.Bind(updateCmd, &updateCommand{files: &updateFiles}, options)

	var commitMessage string
	commitCmd := &cobra.Command{
		Use:   "commit [automation]",
		Short: "Commit staged changes",
		Args:  cobra.MaximumNArgs(1),
	}
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "commit message")
	_ = commitCmd.MarkFlagRequired("message")
	core.Bind(commitCmd, &commitCommand{message: &commitMessage}, options)

	root.AddCommand(statusCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(commitCmd)

	return root
}
