package tasks

import (
	"github.com/spf13/cobra"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

func newChecksCommand(options core.BindOptions) *cobra.Command {
	checksCmd := &cobra.Command{
		Use:     "checks",
		Short:   "Manage task checks",
		Aliases: []string{"check"},
	}

	var (
		listWorkspace string
		listTaskID    string
	)
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List checks on a task",
		Long: `List all checks on a task, with their score, level, and source.

Example:
  superplane tasks checks list --task 1823`,
		Args: cobra.NoArgs,
	}
	listCmd.Flags().StringVar(&listWorkspace, "workspace", "", "workspace key or UUID (default: active workspace)")
	bindTaskIDFlag(listCmd, &listTaskID)
	_ = listCmd.MarkFlagRequired("task")
	core.Bind(listCmd, &checkListCommand{workspace: &listWorkspace, taskID: &listTaskID}, options)

	var (
		addWorkspace string
		addTaskID    string
		addKey       string
		addName      string
		addScore     float64
		addMaxScore  float64
		addLevel     string
		addReason    string
	)
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add or update a check on a task",
		Long: `Add a check to a task. A check with the same key is replaced.
You cannot use the key of a check that an automation owns.

--level is one of: positive, caution, critical.

Example:
  superplane tasks checks add \
    --task 1823 \
    --key tests-pass \
    --name "Tests pass" \
    --score 9 --max-score 10 \
    --level positive \
    --reason "All unit tests pass."`,
		Args: cobra.NoArgs,
	}
	addCmd.Flags().StringVar(&addWorkspace, "workspace", "", "workspace key or UUID (default: active workspace)")
	bindTaskIDFlag(addCmd, &addTaskID)
	addCmd.Flags().StringVar(&addKey, "key", "", "check key")
	addCmd.Flags().StringVar(&addName, "name", "", "check name")
	addCmd.Flags().Float64Var(&addScore, "score", 0, "check score")
	addCmd.Flags().Float64Var(&addMaxScore, "max-score", 0, "maximum score")
	addCmd.Flags().StringVar(&addLevel, "level", "", "check level: positive, caution, or critical")
	addCmd.Flags().StringVar(&addReason, "reason", "", "why the check has this score")
	for _, name := range []string{"task", "key", "name", "score", "max-score", "level", "reason"} {
		_ = addCmd.MarkFlagRequired(name)
	}
	core.Bind(addCmd, &checkAddCommand{
		workspace: &addWorkspace,
		taskID:    &addTaskID,
		key:       &addKey,
		name:      &addName,
		score:     &addScore,
		maxScore:  &addMaxScore,
		level:     &addLevel,
		reason:    &addReason,
	}, options)

	var (
		removeWorkspace string
		removeTaskID    string
		removeKeys      []string
	)
	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove checks from a task",
		Long: `Remove checks that you or an agent added. Checks that an automation
owns are not removed.

Example:
  superplane tasks checks remove --task 1823 --key tests-pass`,
		Args: cobra.NoArgs,
	}
	removeCmd.Flags().StringVar(&removeWorkspace, "workspace", "", "workspace key or UUID (default: active workspace)")
	bindTaskIDFlag(removeCmd, &removeTaskID)
	removeCmd.Flags().StringSliceVar(&removeKeys, "key", nil, "check key (repeat for more keys)")
	_ = removeCmd.MarkFlagRequired("task")
	_ = removeCmd.MarkFlagRequired("key")
	core.Bind(removeCmd, &checkRemoveCommand{
		workspace: &removeWorkspace,
		taskID:    &removeTaskID,
		keys:      &removeKeys,
	}, options)

	checksCmd.AddCommand(listCmd)
	checksCmd.AddCommand(addCmd)
	checksCmd.AddCommand(removeCmd)
	return checksCmd
}
