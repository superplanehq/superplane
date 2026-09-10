package tasks

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "tasks",
		Short:   "Create, inspect, and manage tasks",
		Aliases: []string{"task"},
	}

	var (
		taskListWorkspace  string
		taskListAssignees  []string
		taskListStates     []string
		taskListResults    []string
		taskListUnassigned bool
	)

	taskListCmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a workspace",
		Long: `List tasks in a workspace.

--workspace is a workspace name or UUID. When omitted, the active workspace
from "superplane workspace active" is used.

--assignees accepts user UUIDs or emails (comma-separated or repeated).
--state and --result accept the proto enum tokens (e.g. STATE_OPEN,
RESULT_COMPLETED) or short case-insensitive names (open, draft, closed /
completed, rejected, failed).

By default, only open tasks are shown. Pass --state all to see
draft, open, and closed tasks.

Examples:
  superplane tasks list --workspace shipping --state open
  superplane tasks list --assignees alice@example.com --result failed
  superplane tasks list --unassigned
  superplane tasks list --state all`,
		Args: cobra.NoArgs,
	}
	taskListCmd.Flags().StringVar(&taskListWorkspace, "workspace", "", "workspace name or UUID (default: active workspace)")
	taskListCmd.Flags().StringSliceVar(&taskListAssignees, "assignees", nil, "filter by assignee user UUID or email (repeatable)")
	taskListCmd.Flags().StringSliceVar(&taskListStates, "state", nil, "filter by task state (repeatable, e.g. open or STATE_OPEN); defaults to open when omitted; pass 'all' to include every state")
	taskListCmd.Flags().StringSliceVar(&taskListResults, "result", nil, "filter by task result (repeatable, e.g. completed or RESULT_COMPLETED)")
	taskListCmd.Flags().BoolVar(&taskListUnassigned, "unassigned", false, "only show tasks with no assignees")
	core.Bind(taskListCmd, &taskListCommand{
		workspace:  &taskListWorkspace,
		assignees:  &taskListAssignees,
		states:     &taskListStates,
		results:    &taskListResults,
		unassigned: &taskListUnassigned,
	}, options)

	var (
		taskDescribeWorkspace string
		taskDescribeTaskID    string
	)

	taskDescribeCmd := &cobra.Command{
		Use:   "describe",
		Short: "Show a task's details, comments, and event timeline",
		Long: `Show a task's title, assignees, description, comments, and
timeline of events.

--workspace is a workspace name or UUID. When omitted, the active workspace
from "superplane workspace active" is used. --task is the task UUID or slug
(e.g. prefix-123).

Example:
  superplane tasks describe --workspace shipping --task "$TID"`,
		Args: cobra.NoArgs,
	}
	taskDescribeCmd.Flags().StringVar(&taskDescribeWorkspace, "workspace", "", "workspace name or UUID (default: active workspace)")
	bindTaskIDFlag(taskDescribeCmd, &taskDescribeTaskID)
	core.Bind(taskDescribeCmd, &taskDescribeCommand{
		workspace: &taskDescribeWorkspace,
		taskID:    &taskDescribeTaskID,
	}, options)

	var (
		taskCreateWorkspace   string
		taskCreateTitle       string
		taskCreateDescription string
		taskCreateFile        string
		taskCreateAssignees   []string
	)

	taskCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a task",
		Long: `Create a task in draft state.

--workspace is a workspace name or UUID. When omitted, the active workspace
from "superplane workspace active" is used. --title is required.
--description sets the description inline; -f/--file reads it from a
file (or - for stdin) instead; the two are mutually exclusive.

--assignee accepts a user UUID or email and is repeatable. When omitted,
the task is assigned to the user running the command.

Examples:
  superplane tasks create --title "Ship the feature" --description "..."

  superplane tasks create \
    --title "Ship the feature" \
    -f ./description.md \
    --assignee alice@example.com \
    --assignee bob@example.com`,
		Args: cobra.NoArgs,
	}
	taskCreateCmd.Flags().StringVar(&taskCreateWorkspace, "workspace", "", "workspace name or UUID (default: active workspace)")
	taskCreateCmd.Flags().StringVar(&taskCreateTitle, "title", "", "task title (required)")
	taskCreateCmd.Flags().StringVar(&taskCreateDescription, "description", "", "task description (inline)")
	taskCreateCmd.Flags().StringVarP(&taskCreateFile, "file", "f", "", "read description from file (or - for stdin)")
	taskCreateCmd.Flags().StringArrayVar(&taskCreateAssignees, "assignee", nil, "assignee user UUID or email (repeatable); defaults to the current user when omitted")
	core.Bind(taskCreateCmd, &taskCreateCommand{
		workspace:   &taskCreateWorkspace,
		title:       &taskCreateTitle,
		description: &taskCreateDescription,
		file:        &taskCreateFile,
		assignees:   &taskCreateAssignees,
	}, options)

	var (
		taskDispatchWorkspace string
		taskDispatchTaskID    string
		taskDispatchLine      string
	)

	taskDispatchCmd := &cobra.Command{
		Use:   "dispatch",
		Short: "Dispatch a task to a workspace line",
		Long: `Dispatch a task to a workspace line, starting its execution.

--workspace is a workspace name or UUID. When omitted, the active workspace
from "superplane workspace active" is used. --task is the task UUID or slug
(e.g. prefix-123). --line is the target workspace line's name.

A draft task moves to the open state on its first dispatch.

Example:
  superplane tasks dispatch --task "$TID" --line build`,
		Args: cobra.NoArgs,
	}
	taskDispatchCmd.Flags().StringVar(&taskDispatchWorkspace, "workspace", "", "workspace name or UUID (default: active workspace)")
	bindTaskIDFlag(taskDispatchCmd, &taskDispatchTaskID)
	taskDispatchCmd.Flags().StringVar(&taskDispatchLine, "line", "", "workspace line name (required)")
	core.Bind(taskDispatchCmd, &taskDispatchCommand{
		workspace: &taskDispatchWorkspace,
		taskID:    &taskDispatchTaskID,
		line:      &taskDispatchLine,
	}, options)

	var (
		taskAssignWorkspace string
		taskAssignTaskID    string
		taskAssignAssignees []string
	)

	taskAssignCmd := &cobra.Command{
		Use:   "assign",
		Short: "Set a task's assignees",
		Long: `Set a task's assignees.

--workspace is a workspace name or UUID. When omitted, the active workspace
from "superplane workspace active" is used. --task is the task UUID or slug
(e.g. prefix-123).

--assignee accepts a user UUID or email and is repeatable; at least one is
required. This command replaces the entire assignee list with the ones
given — it does not add to the existing list.

Example:
  superplane tasks assign --task "$TID" --assignee alice@example.com --assignee bob@example.com`,
		Args: cobra.NoArgs,
	}
	taskAssignCmd.Flags().StringVar(&taskAssignWorkspace, "workspace", "", "workspace name or UUID (default: active workspace)")
	bindTaskIDFlag(taskAssignCmd, &taskAssignTaskID)
	taskAssignCmd.Flags().StringArrayVar(&taskAssignAssignees, "assignee", nil, "assignee user UUID or email (repeatable, required); replaces the full assignee list")
	core.Bind(taskAssignCmd, &taskAssignCommand{
		workspace: &taskAssignWorkspace,
		taskID:    &taskAssignTaskID,
		assignees: &taskAssignAssignees,
	}, options)

	artifactCmd := &cobra.Command{
		Use:     "artifacts",
		Short:   "Manage task artifacts",
		Aliases: []string{"artifact"},
	}

	var (
		artifactAddWorkspace string
		artifactAddTaskID    string
		artifactAddTypeName  string
		artifactAddTitle     string
		artifactAddBody      string
		artifactAddFile      string
		artifactAddURL       string
		artifactAddName      string
	)

	artifactAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Attach an artifact to a task",
		Long: `Attach a typed artifact to a task.

--workspace is a workspace name or UUID. When omitted, the active workspace
from "superplane workspace active" is used. --task is the task UUID or slug.
--type is one of: markdown, branch, link.

Examples:
  superplane tasks artifacts list --workspace shipping --task "$TID"

  # Uses the active workspace
  superplane tasks artifacts add \
    --task "$TID" \
    --type markdown \
    --title "PLAN.md" \
    -f ./PLAN.md

  superplane tasks artifacts add \
    --workspace shipping \
    --task "$TID" \
    --type markdown \
    --title "PLAN.md" \
    -f ./PLAN.md

  superplane tasks artifacts add \
    --task "$TID" \
    --type branch \
    --name feature/login

  superplane tasks artifacts add \
    --task "$TID" \
    --type link \
    --url https://preview.example.com/pr-42 \
    --title Preview`,
		Args: cobra.NoArgs,
	}
	artifactAddCmd.Flags().StringVar(&artifactAddWorkspace, "workspace", "", "workspace name or UUID (default: active workspace)")
	artifactAddCmd.Flags().StringVar(&artifactAddTaskID, "task", "", "task UUID or slug (e.g. prefix-123)")
	artifactAddCmd.Flags().StringVar(&artifactAddTypeName, "type", "", "artifact type: markdown, branch, or link")
	artifactAddCmd.Flags().StringVar(&artifactAddTitle, "title", "", "artifact title")
	artifactAddCmd.Flags().StringVar(&artifactAddBody, "body", "", "markdown body (inline)")
	artifactAddCmd.Flags().StringVarP(&artifactAddFile, "file", "f", "", "read markdown body from file (or - for stdin)")
	artifactAddCmd.Flags().StringVar(&artifactAddURL, "url", "", "artifact URL (required for link)")
	artifactAddCmd.Flags().StringVar(&artifactAddName, "name", "", "branch name (required for branch)")
	_ = artifactAddCmd.MarkFlagRequired("task")
	_ = artifactAddCmd.MarkFlagRequired("type")
	core.Bind(artifactAddCmd, &artifactAddCommand{
		workspace: &artifactAddWorkspace,
		taskID:    &artifactAddTaskID,
		typ:       &artifactAddTypeName,
		title:     &artifactAddTitle,
		body:      &artifactAddBody,
		file:      &artifactAddFile,
		url:       &artifactAddURL,
		name:      &artifactAddName,
	}, options)

	var (
		artifactListWorkspace string
		artifactListTaskID    string
	)
	artifactListCmd := &cobra.Command{
		Use:   "list",
		Short: "List artifacts on a task",
		Long: `List artifacts on a task.

--workspace is a workspace name or UUID. When omitted, the active workspace
from "superplane workspace active" is used.

Example:
  superplane tasks artifacts list --workspace shipping --task "$TID"`,
		Args: cobra.NoArgs,
	}
	artifactListCmd.Flags().StringVar(&artifactListWorkspace, "workspace", "", "workspace name or UUID (default: active workspace)")
	artifactListCmd.Flags().StringVar(&artifactListTaskID, "task", "", "task UUID or slug (e.g. prefix-123)")
	_ = artifactListCmd.MarkFlagRequired("task")
	core.Bind(artifactListCmd, &artifactListCommand{
		workspace: &artifactListWorkspace,
		taskID:    &artifactListTaskID,
	}, options)

	artifactCmd.AddCommand(artifactAddCmd)
	artifactCmd.AddCommand(artifactListCmd)

	root.AddCommand(taskListCmd)
	root.AddCommand(taskDescribeCmd)
	root.AddCommand(taskCreateCmd)
	root.AddCommand(taskDispatchCmd)
	root.AddCommand(taskAssignCmd)
	root.AddCommand(artifactCmd)

	return root
}

func bindTaskIDFlag(cmd *cobra.Command, dest *string) {
	cmd.Flags().StringVar(dest, "task", "", "task UUID or slug (e.g. prefix-123)")
}
