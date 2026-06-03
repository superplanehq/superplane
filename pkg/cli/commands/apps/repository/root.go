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

	commitCmd := &cobra.Command{
		Use:   "commit [app-name-or-id]",
		Short: "Commit files to a draft branch",
		Long: `Atomically commit one or more files to a draft git branch.

Use --path repeatedly to include files. When --branch is omitted, the current
user's default draft branch is used (created if needed).`,
		Args: cobra.MaximumNArgs(1),
	}
	var (
		commitBranch          string
		commitExpectedHeadSHA string
		commitMessage         string
		commitPaths           []string
	)
	commitCmd.Flags().StringVar(&commitBranch, "branch", "", "draft branch to commit to")
	commitCmd.Flags().StringVar(&commitExpectedHeadSHA, "expected-head-sha", "", "required head SHA for optimistic concurrency")
	commitCmd.Flags().StringVar(&commitMessage, "message", "Update repository files", "git commit message")
	commitCmd.Flags().StringArrayVar(&commitPaths, "path", nil, "local file path to commit (repeatable)")
	_ = commitCmd.MarkFlagRequired("path")
	core.Bind(commitCmd, &CommitCommand{
		branch:          &commitBranch,
		expectedHeadSHA: &commitExpectedHeadSHA,
		message:         &commitMessage,
		paths:           &commitPaths,
	}, options)
	root.AddCommand(commitCmd)

	return root
}
